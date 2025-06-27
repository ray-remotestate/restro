package handlers

import (
	"time"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"github.com/ray-remotestate/restro/config"
	"github.com/ray-remotestate/restro/database"
	"github.com/ray-remotestate/restro/database/dbhelper"
	"github.com/ray-remotestate/restro/middlewares"
	"github.com/ray-remotestate/restro/models"
	"github.com/ray-remotestate/restro/utils"
)

func Register(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "password must be at least 6 characters", http.StatusBadRequest)
	}

	exists, err := dbhelper.IsUserExists(req.Email)
	if err != nil {
		http.Error(w, "failed to check user existence", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "user already exists", http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	var userID uuid.UUID
	var accToken, refToken string
	txErr := database.Tx(func(tx *sql.Tx) error { // using *sql.Tx instead *sql.DB as we want both the operation to either commit together or fail together.
		userID, err = dbhelper.CreateUser(tx, req.Name, req.Email, hashedPassword)
		if err != nil {
			logrus.Printf("failed to create user, error: %v", err)
			return err
		}

		err = dbhelper.AssignRole(tx, userID, models.RoleUser)
		if err != nil {
			logrus.Printf("failed to assign role to the user, error: %v", err)
			return err
		}

		accToken, refToken, err = utils.GenerateTokens(userID, []string{string(models.RoleUser)})
		if err != nil {
			logrus.Printf("failed to generate token, error: %v", err)
			return err
		}

		return nil
	})
	if txErr != nil {
		http.Error(w, "failed to register user", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"user_id": userID,
		"email":   req.Email,
		"name":    req.Name,
		"access_token":   accToken,
		"refersh_token": refToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func RefershToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token missing", http.StatusUnauthorized)
		return
	}
	refreshToken := cookie.Value

	claims := &middlewares.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.SecretKey), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	newAccessToken, newRefreshToken, err := utils.GenerateTokens(claims.UserID, claims.Roles)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	resp := map[string]string{
		"access_token": newAccessToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Login(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	userID, name, err := dbhelper.GetUserByPassword(req.Email, req.Password)
	if err == sql.ErrNoRows {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	rows, err := dbhelper.GetUserRoleByUserID(userID)
	if err != nil {
		http.Error(w, "could not fetch roles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err == nil {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		http.Error(w, "no roles assigned", http.StatusForbidden)
		return
	}

	accessToken, refreshToken, err := utils.GenerateTokens(userID, roles)
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	resp := map[string]interface{}{
		"user_id":      userID,
		"name":         name,
		"email":        req.Email,
		"access_token": accessToken,
		"roles":        roles,
		"message":		"Successfully logged in",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Unix(0, 0), // Expire immediately
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully logged out",
	})
}

func CreateSubAdmin(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		http.Error(w, "name and email are required", http.StatusBadRequest)
		return
	}

	userID, err := dbhelper.GetUserByEmail(req.Email)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if err == sql.ErrNoRows {
		http.Error(w, "user does not exist", http.StatusInternalServerError)
		// User does not exist — create new one
		// hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		// if err != nil {
		// 	http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		// 	return
		// }
		// err = db.QueryRow(`
		// 	INSERT INTO users (name, email, password)
		// 	VALUES ($1, $2, $3)
		// 	RETURNING id
		// `, req.Name, req.Email, string(hashedPassword)).Scan(&userID)
		// if err != nil {
		// 	http.Error(w, "Failed to create user", http.StatusInternalServerError)
		// 	return
		// }
		return
	}

	isSubAdmin, err := dbhelper.IsSubAdmin(userID)
	if err != nil {
		http.Error(w, "role check failed", http.StatusInternalServerError)
		return
	}
	if isSubAdmin {
		http.Error(w, "user is already a subadmin", http.StatusConflict)
		return
	}

	err = dbhelper.MakeSubAdmin(userID)
	if err != nil {
		http.Error(w, "failed to assign subadmin role", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Subadmin created successfully",
		"user_id": userID.String(),
	})
}

func ListSubAdmins(w http.ResponseWriter, r *http.Request) {
	type SubAdmin struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	}

	rows, err := dbhelper.ListAllSubadmins()
	if err != nil {
		http.Error(w, "Failed to query subadmins", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var subadmins []SubAdmin
	for rows.Next() {
		var sa SubAdmin
		if err := rows.Scan(&sa.ID, &sa.Name, &sa.Email); err != nil {
			http.Error(w, "Failed to parse result", http.StatusInternalServerError)
			return
		}
		subadmins = append(subadmins, sa)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error reading results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subadmins)
}

func ListAllUsersBySubAdmin(w http.ResponseWriter, r *http.Request) {
	type User struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	}

	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	roles, ok := r.Context().Value("roles").([]string)
	if !ok || len(roles) == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	isAdmin := false
	for _, role := range roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = database.Restro.Query(`
			SELECT id, name, email
			FROM users
			WHERE archived_at IS NULL
		`)
	} else {
		rows, err = database.Restro.Query(`
			SELECT id, name, email
			FROM users
			WHERE created_by = $1 AND archived_at IS NULL
		`, userID)
	}

	if err != nil {
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			http.Error(w, "Failed to read data", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Row error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func AddAddress(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	type Input struct {
		Address   string  `json:"address"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON input", http.StatusBadRequest)
		return
	}

	if input.Address == "" {
		http.Error(w, "address is required", http.StatusBadRequest)
		return
	}

	var addressID uuid.UUID
	err := database.Restro.QueryRow(`
		INSERT INTO addresses (user_id, address, latitude, longitude)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, userID, input.Address, input.Latitude, input.Longitude).Scan(&addressID)
	if err != nil {
		http.Error(w, "failed to add address", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":    "Address added successfully",
		"address_id": addressID.String(),
	})
}