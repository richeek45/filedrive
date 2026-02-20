package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/richeek45/filedrive/models"
	"github.com/richeek45/filedrive/repositories"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthController struct {
	Repo             *repositories.UserRepository
	oauthConfig      *oauth2.Config
	oauthStateString string
}

func NewAuthController(userRepo *repositories.UserRepository) *AuthController {
	return &AuthController{
		Repo: userRepo,
		oauthConfig: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (r *AuthController) generateStateOauthCookie() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// GoogleLogin redirects to Google's OAuth2 consent page
func (r *AuthController) GoogleLogin(c *gin.Context) {
	state := r.generateStateOauthCookie()
	c.SetCookie("oauth_state", state, 3600, "/", "localhost", false, true)
	url := r.oauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (r *AuthController) GoogleCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, _ := c.Cookie("oauth_state")

	if state != cookieState {
		c.Redirect(http.StatusTemporaryRedirect, "http://localhost:5173/login?error=invalid_state")
		return
	}

	code := c.Query("code")

	token, err := r.oauthConfig.Exchange(c, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange token"})
		return
	}

	userInfo, err := r.getUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	// Map Google info to our Struct
	// Note: We don't manually set ID or CreatedAt because gorm.Model handles it
	user := &models.User{
		GoogleID:    userInfo["id"].(string),
		Email:       userInfo["email"].(string),
		FirstName:   userInfo["given_name"].(string),  // Google provides given_name
		LastName:    userInfo["family_name"].(string), // Google provides family_name
		Picture:     userInfo["picture"].(string),
		LastLoginAt: time.Now(),
	}

	if err := r.Repo.UpsertByGoogleID(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync user data"})
	}

	tokens, err := r.generateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	redirectURL := frontendURL + "/oauth-callback?access_token=" + tokens.AccessToken +
		"&refresh_token=" + tokens.RefreshToken +
		"&expires_in=" + string(rune(tokens.ExpiresIn))

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (r *AuthController) getUserInfo(accessToken string) (map[string]interface{}, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (r *AuthController) generateTokens(user *models.User) (*models.TokenDetails, error) {
	accessTokenExpiry := time.Now().Add(time.Hour * 1) // 1 hour
	accessClaims := &models.Claims{
		UserID: strconv.FormatUint(uint64(user.ID), 10),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "your-app",
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return nil, err
	}

	refreshTokenExpiry := time.Now().Add(time.Hour * 24 * 7) // 7 days
	refreshClaims := &models.Claims{
		UserID: strconv.FormatUint(uint64(user.ID), 10),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshTokenExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "your-app",
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return nil, err
	}

	return &models.TokenDetails{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    accessTokenExpiry.Unix(),
	}, nil

}

func (r *AuthController) RefreshToken(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Verify refresh token
	token, err := jwt.ParseWithClaims(request.RefreshToken, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	claims, ok := token.Claims.(*models.Claims)
	if !ok || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	id64, err := strconv.ParseUint(claims.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format in token"})
		return
	}
	uintID := uint(id64)

	var user models.User
	err = r.Repo.DB.First(&user, uintID).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User Not Found"})
	}

	// Generate new tokens
	tokens, err := r.generateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, tokens)

}

func (r *AuthController) Logout(c *gin.Context) {
	// In a stateless JWT setup, you might want to:
	// 1. Add the token to a blacklist (if you implement one)
	// 2. Or simply let the client delete the token

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
