package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/dtos"
	"github.com/richeek45/filedrive/models"
	"github.com/richeek45/filedrive/repositories"
	"github.com/wneessen/go-mail"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FileController struct {
	Repo       *repositories.FileRepository
	FolderRepo *repositories.FolderRepository
	UserRepo   *repositories.UserRepository
	S3Client   *s3.Client
	Bucket     string
}

var (
	folderCache     = sync.Map{}
	cleanupInterval = 5 * time.Minute
)

type CacheEntry struct {
	folderID  uuid.UUID
	expiresAt time.Time
}

func getFolderCacheKey(userID uuid.UUID, rootID *uuid.UUID, path string) string {
	rootStr := "root"
	if rootID != nil {
		rootStr = rootID.String()
	}
	return fmt.Sprintf("%s:%s:%s", userID.String(), rootStr, path)
}

func StartCacheCleaner() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			folderCache.Range(func(key, value any) bool {
				entry := value.(CacheEntry)
				if now.After(entry.expiresAt) {
					folderCache.Delete(key)
				}
				return true
			})
		}
	}()
}

func (fc *FileController) GetFilesFromParentFolder(c *gin.Context) {
	var req struct {
		FolderID string `form:"parentId"`
		IsTrash  bool   `form:"isTrash"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := uuid.MustParse(c.GetString("userID"))
	var folderIDPtr *uuid.UUID
	if req.FolderID != "" {
		parsed := uuid.MustParse(req.FolderID)
		folderIDPtr = &parsed
	}
	files, err := fc.Repo.GetFiles(userID, folderIDPtr, req.IsTrash)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []dtos.FileResponse

	for _, f := range files {

		response = append(response, dtos.FileResponse{
			ID:           f.ID,
			Name:         f.Name,
			Size:         f.Size,
			MimeType:     f.MimeType,
			CreatedAt:    f.CreatedAt,
			IsDeleted:    f.IsDeleted,
			UploadStatus: f.UploadStatus,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (fc *FileController) GetDownloadURL(c *gin.Context) {
	fileID := c.Param("fileId")
	userID := c.GetString("userID")

	file, err := fc.Repo.GetFileByID(uuid.MustParse(fileID), uuid.MustParse(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// 1. URL-encode the filename to handle spaces and special characters
	// PathEscape is better here than QueryEscape as it handles spaces as %20
	encodedName := url.PathEscape(file.Name)

	// 2. Use the RFC 6266 format: filename*=UTF-8''{encoded_name}
	contentDisposition := fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName)

	// 2. Create Presigned URL (Valid for 15 minutes)
	presignClient := s3.NewPresignClient(fc.S3Client)
	presignedReq, err := presignClient.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
		Bucket:                     aws.String(fc.Bucket),
		Key:                        aws.String(file.ObjectKey),
		ResponseContentDisposition: aws.String(contentDisposition),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": presignedReq.URL})
}

func (fc *FileController) MoveToTrash(c *gin.Context) {
	fileId, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := uuid.MustParse(c.GetString("userID"))
	if err := fc.Repo.DeleteFile(fileId, userId, fc.S3Client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "File moved to trash"})
}

func (fc *FileController) RenameFile(c *gin.Context) {
	var req struct {
		NewName string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileID := uuid.MustParse(c.Param("fileId"))
	userID := uuid.MustParse(c.GetString("userID"))

	err := fc.Repo.DB.Model(&models.File{}).
		Where("id = ? AND owner_id = ?", fileID, userID).
		Update("name", req.NewName).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Renamed successfully"})
}

func (fc *FileController) InitiateMultiPartUpload(c *gin.Context) {
	var req struct {
		FileName     string     `json:"fileName" binding:"required"`
		ContentType  string     `json:"contentType" binding:"required"`
		Size         int64      `json:"size" binding:"required"`
		ParentID     *uuid.UUID `json:"parentId"`
		TotalChunks  *int       `json:"totalChunks" binding:"required"`
		RelativePath string     `json:"relativePath"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := uuid.MustParse(c.GetString("userID"))
	finalParentID := req.ParentID

	user, err := fc.UserRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if user.StorageUsed+req.Size >= user.StorageLimit {
		c.JSON(http.StatusInsufficientStorage, gin.H{"error": "Not enough space. Delete some files"})
		return
	}

	// 1. Check DB for existing upload for this User + FileName + ParentID
	var pending models.PendingUpload
	query := fc.Repo.DB.Where("user_id = ? AND file_name = ?", userID, req.FileName)
	if req.ParentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", req.ParentID)
	}
	err = query.First(&pending).Error

	if err == nil {
		// Found existing! Ask S3 which parts it already has
		out, S3err := fc.S3Client.ListParts(c.Request.Context(), &s3.ListPartsInput{
			Bucket:   aws.String(fc.Bucket),
			Key:      aws.String(pending.S3Key),
			UploadId: aws.String(pending.UploadID),
		})

		if S3err != nil {
			// If S3 says it doesn't exist (maybe expired), delete pending and start fresh
			fc.Repo.DB.Delete(&pending)
		} else {
			// Return parts with ETags so frontend can "Complete" later
			c.JSON(http.StatusOK, gin.H{
				"uploadId":       pending.UploadID,
				"key":            pending.S3Key,
				"completedParts": out.Parts, // This includes PartNumber and ETag
				"resumed":        true,
			})
			return
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("response not found")
	}

	key := fmt.Sprintf("uploads/%s/%s", uuid.New().String(), req.FileName)

	input := &s3.CreateMultipartUploadInput{
		Bucket:      &fc.Bucket,
		Key:         aws.String(key),
		ContentType: aws.String(req.ContentType),
	}

	resp, err := fc.S3Client.CreateMultipartUpload(c.Request.Context(), input)

	fmt.Println(err)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate upload"})
		return
	}

	if req.RelativePath != "" {
		pathsSplit := strings.Split(req.RelativePath, "/")

		folderParts := pathsSplit[:len(pathsSplit)-1]

		if len(folderParts) > 0 {
			currentParentID := req.ParentID
			accumulatedPath := ""

			for _, folderName := range folderParts {
				accumulatedPath := filepath.Join(accumulatedPath, folderName)
				cacheKey := getFolderCacheKey(userID, req.ParentID, accumulatedPath)

				if val, ok := folderCache.Load(cacheKey); ok {
					entry := val.(CacheEntry)
					if time.Now().Before(entry.expiresAt) {
						id := entry.folderID
						currentParentID = &id
						continue
					}
				}

				newFolderID, err := fc.FolderRepo.EnsureFolderExists(userID, currentParentID, folderName)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Folder creation failed"})
					return
				}

				folderCache.Store(cacheKey, CacheEntry{
					folderID:  newFolderID,
					expiresAt: time.Now().Add(10 * time.Minute),
				})

				currentParentID = &newFolderID
			}
			finalParentID = currentParentID
		}

	}

	newFile := &models.File{
		Name:         req.FileName,
		OwnerID:      userID,
		FolderID:     finalParentID,
		Size:         req.Size,
		MimeType:     &req.ContentType,
		BucketName:   fc.Bucket,
		ObjectKey:    key,
		S3UploadID:   resp.UploadId,
		UploadStatus: "pending",
		TotalChunks:  req.TotalChunks,
	}

	pendingEntry := &models.PendingUpload{
		ID:         uuid.New(),
		UserID:     userID,
		UploadID:   *resp.UploadId,
		S3Key:      key,
		FileName:   req.FileName,
		ParentID:   req.ParentID,
		TotalParts: *req.TotalChunks,
	}

	if err := fc.Repo.UpsertFilePending(newFile, pendingEntry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploadId": *resp.UploadId,
		"key":      key,
	})
}

func (fc *FileController) PresignPart(c *gin.Context) {
	var req struct {
		UploadID   string `json:"uploadId" binding:"required"`
		Key        string `json:"key" binding:"required"`
		PartNumber int32  `json:"partNumber" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	presignClient := s3.NewPresignClient(fc.S3Client)

	// Request a presigned URL for the UploadPart operation
	presignedReq, err := presignClient.PresignUploadPart(c.Request.Context(), &s3.UploadPartInput{
		Bucket:     aws.String(fc.Bucket),
		Key:        aws.String(req.Key),
		UploadId:   aws.String(req.UploadID),
		PartNumber: aws.Int32(req.PartNumber),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to presign part"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": presignedReq.URL})
}

func (fc *FileController) CompleteMultipartUpload(c *gin.Context) {
	var req struct {
		UploadID string                `json:"uploadId" binding:"required"`
		Key      string                `json:"key" binding:"required"`
		ParentID *uuid.UUID            `json:"parentId"`
		Parts    []types.CompletedPart `json:"parts" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := fc.S3Client.CompleteMultipartUpload(c.Request.Context(), &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(fc.Bucket),
		Key:      aws.String(req.Key),
		UploadId: aws.String(req.UploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: req.Parts,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete S3 upload"})
		return
	}

	finalETag := ""
	if result.ETag != nil {
		finalETag = *result.ETag
	}

	err = fc.Repo.FinalizeFile(req.UploadID, len(req.Parts), finalETag, "completed")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update file record"})
		return
	}
	fc.Repo.DB.Where("upload_id = ?", req.UploadID).Delete(&models.PendingUpload{})

	c.JSON(http.StatusOK, gin.H{"message": "upload completed successfully"})
}

// Syncs the files on opening my files tab to check and update the upload status of pending S3 multipart uploads
func (fc *FileController) SyncUserUploads(c *gin.Context) {
	userID := uuid.MustParse(c.GetString("userID"))

	var pendingFiles []models.File
	// Only sync files that aren't 'completed' or 'error'
	fc.Repo.DB.Where("owner_id = ? AND upload_status IN ?", userID, []string{"pending", "uploading", "paused"}).Find(&pendingFiles)

	for _, file := range pendingFiles {
		fmt.Println(file.Name)
		if file.S3UploadID == nil {
			continue
		}
		// if file.S3UploadID == nil || time.Since(file.UpdatedAt) < 5 * time.Minute { continue }

		out, err := fc.S3Client.ListParts(c.Request.Context(), &s3.ListPartsInput{
			Bucket:   aws.String(file.BucketName),
			Key:      aws.String(file.ObjectKey),
			UploadId: file.S3UploadID,
		})

		if err != nil {
			var apiErr smithy.APIError
			// Check if the error is specifically because the upload no longer exists in S3
			if errors.As(err, &apiErr) {
				switch apiErr.ErrorCode() {
				case "NoSuchUpload":
					// S3 Lifecycle rule likely deleted the parts. Reset the DB record.
					fc.Repo.DB.Model(&file).Updates(map[string]interface{}{
						"s3_upload_id":          nil,
						"uploaded_chunks":       0,
						"uploaded_part_numbers": 0,
						"upload_status":         "paused",
					})
					continue
				}
			}
			continue
		}

		// Update DB with what S3 actually has
		fc.Repo.DB.Model(&file).Updates(map[string]interface{}{
			"uploaded_chunks": len(out.Parts),
			"upload_status":   "paused",
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sync complete"})
}

type ShareRequest struct {
	FileID     uuid.UUID             `json:"fileId" binding:"required"`
	FolderID   uuid.UUID             `json:"folderId"`
	Emails     []string              `json:"emails" binding:"required"`
	Permission models.PermissionType `json:"permission" binding:"required"`
}

func (fc *FileController) ShareFilesToUsersByEmails(c *gin.Context) {
	var req ShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := uuid.MustParse(c.GetString("userID"))

	fmt.Println(userID, req.FileID)

	file, err := fc.Repo.GetFileByID(req.FileID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	var targetUsers []models.Users
	if err := fc.UserRepo.DB.Where("email IN ?", req.Emails).Find(&targetUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to look up users"})
		return
	}

	if len(targetUsers) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No registered users found for these emails"})
		return
	}

	var permissions []models.ResourcePermission
	for _, user := range targetUsers {
		permission := models.ResourcePermission{
			FileID:     &file.ID,
			UserID:     user.ID,
			FolderID:   &req.FolderID,
			GrantedBy:  userID,
			Permission: models.PermissionType(req.Permission),
			CreatedAt:  time.Now(),
		}

		if req.FolderID != uuid.Nil {
			permission.FolderID = &req.FolderID
		} else {
			permission.FolderID = nil
		}
		permissions = append(permissions, permission)
	}

	err = fc.Repo.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "file_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"permission", "granted_by"}),
	}).Create(&permissions).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permissions"})
		return
	}

	go fc.sendShareEmails(targetUsers, file.Name)

	c.JSON(http.StatusOK, gin.H{
		"message":         "File shared successfully",
		"sharedWithCount": len(targetUsers),
	})
}

func (fc *FileController) sendShareEmails(users []models.Users, fileName string) {
	m := mail.NewMsg()
	if err := m.From(os.Getenv("GMAIL_USER")); err != nil {
		fmt.Printf("failed to set from address: %v\n", err)
		return
	}

	client, err := mail.NewClient("smtp.gmail.com",
		mail.WithPort(465),
		mail.WithSSL(),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(os.Getenv("GMAIL_USER")),
		mail.WithPassword(os.Getenv("GMAIL_APP_PASSWORD")),
	)

	if err != nil {
		fmt.Printf("failed to create mail client: %v\n", err)
		return
	}

	// Need to change the link to real link later
	for _, user := range users {
		m.To(user.Email)
		m.Subject(fmt.Sprintf("A file has been shared with you: %s", fileName))
		body := fmt.Sprintf(`
			<h3>Hello %s,</h3>
			<p>You have been granted access to <b>%s</b>.</p>
			<p>Click the link below to view the file:</p>
			<a href="%s/dashboard/shared">View File</a>
		`, user.FirstName, fileName, os.Getenv("FRONTEND_URL"))

		m.SetBodyString(mail.TypeTextHTML, body)

		if err := client.DialAndSend(m); err != nil {
			fmt.Printf("failed to send email to %s: %v\n", user.Email, err)
		}
	}

}

func (fc *FileController) SharedWithUserFiles(c *gin.Context) {
	userID := uuid.MustParse(c.GetString("userID"))

	files, err := fc.Repo.SharedFilesByUserID(userID)
	if err != nil {
		log.Println("Error: ", err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	}

	var response []dtos.SharedFileResponse

	for _, f := range files {

		response = append(response, dtos.SharedFileResponse{
			FileResponse: dtos.FileResponse{
				ID:        f.ID,
				Name:      f.Name,
				Size:      f.Size,
				MimeType:  f.MimeType,
				CreatedAt: f.CreatedAt,
				IsDeleted: f.IsDeleted,
			},
			Permission: f.Permission,
			SharedBy:   f.SharedBy,
		})
	}

	c.JSON(http.StatusOK, response)
}
