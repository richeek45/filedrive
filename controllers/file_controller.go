package controllers

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richeek45/filedrive/models"
	"github.com/richeek45/filedrive/repositories"
)

type FileController struct {
    Repo     *repositories.FileRepository
    S3Client *s3.Client
    Bucket   string
}

// func (fc *FileController) getPresigner() *s3.PresignClient {
//     return s3.NewPresignClient(fc.S3Client)
// }

func (fc *FileController) GetFilesFromParentFolder(c *gin.Context) {
	var req struct {
		FolderID string `json:"parentId"`
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
	files, err := fc.Repo.GetFiles(userID, folderIDPtr)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, files)
}

func (fc *FileController) InitiateMultiPartUpload(c *gin.Context) {
	var req struct {
        FileName    string `json:"fileName" binding:"required"`
        ContentType string `json:"contentType" binding:"required"`
		Size        int64      `json:"size" binding:"required"`
        ParentID    *uuid.UUID `json:"parentId"`
		TotalChunks *int `json:"totalChunks" binding:"required"`
    }

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := uuid.MustParse(c.GetString("userID"))

	// 1. Check DB for existing upload for this User + FileName + ParentID
    var pending models.PendingUpload
    err := fc.Repo.DB.Where("user_id = ? AND file_name = ? AND parent_id = ?", 
        userID, req.FileName, req.ParentID).First(&pending).Error

    if err != nil {
        // Found existing! Ask S3 which parts it already has
        out, _ := fc.S3Client.ListParts(c.Request.Context(), &s3.ListPartsInput{
            Bucket:   aws.String(fc.Bucket),
            Key:      aws.String(pending.S3Key),
            UploadId: aws.String(pending.UploadID),
        })

		fmt.Println("Parts = ", out.Parts);

        c.JSON(http.StatusOK, gin.H{
            "uploadId":      pending.UploadID,
            "key":           pending.S3Key,
            "completedParts": out.Parts, // Frontend skips these
            "resumed":       true,
        })
        return
    }

	fmt.Println("Uploading to:", fc.Bucket)

    key := fmt.Sprintf("uploads/%s/%s", uuid.New().String(), req.FileName)
	
	input := &s3.CreateMultipartUploadInput{
		Bucket: &fc.Bucket,
		Key: aws.String(key),
		ContentType: aws.String(req.ContentType),
	}

	resp, err := fc.S3Client.CreateMultipartUpload(c.Request.Context(), input)

	fmt.Println(err)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate upload"})
        return
	}

	newFile := &models.File{
        Name:         req.FileName,
        OwnerID:      userID,
        FolderID:     req.ParentID,
        Size:         req.Size,
        MimeType:     &req.ContentType,
        BucketName:   fc.Bucket,
        ObjectKey:    key,
        S3UploadID:   resp.UploadId,
        UploadStatus: "pending",
		TotalChunks: req.TotalChunks,
    }

    if err := fc.Repo.UpsertFilePending(newFile); err != nil {
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
        UploadID string `json:"uploadId" binding:"required"`
        Key      string      `json:"key" binding:"required"`
        ParentID *uuid.UUID  `json:"parentId"`
        Parts    []types.CompletedPart `json:"parts" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 1. Tell S3 to finalize
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

