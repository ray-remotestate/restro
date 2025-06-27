package handlers

import(
	"encoding/json"
	"database/sql"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/gorilla/mux"
	"github.com/ray-remotestate/restro/database"
)

func CreateResource(w http.ResponseWriter, r *http.Request) {
	resourceType := r.URL.Query().Get("type")
	if resourceType == "" {
		http.Error(w, "Missing resource type", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roles, ok := r.Context().Value("roles").([]string)
	if !ok || len(roles) == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isAdmin := slices.Contains(roles, "admin")
	isSubAdmin := slices.Contains(roles, "subadmin")

	if !isAdmin && !isSubAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch resourceType {
	case "user":
		createUser(w, r, userID)
	case "restaurant":
		createRestaurant(w, r, userID)
	case "menu":
		createMenuItem(w, r, userID)
	default:
		http.Error(w, "Invalid resource type", http.StatusBadRequest)
	}
}

func ListRestaurants(w http.ResponseWriter, r *http.Request) {
	type Restaurant struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Latitude    float64   `json:"latitude"`
		Longitude   float64   `json:"longitude"`
		CreatedAt   time.Time `json:"created_at"`
	}

	query := `
		SELECT id, name, description, latitude, longitude, created_at
		FROM restaurants
		ORDER BY created_at DESC
	`

	rows, err := database.Restro.Query(query)
	if err != nil {
		http.Error(w, "failed to query restaurants", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var restaurants []Restaurant
	for rows.Next() {
		var r Restaurant
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Latitude, &r.Longitude, &r.CreatedAt); err != nil {
			http.Error(w, "error reading data", http.StatusInternalServerError)
			return
		}
		restaurants = append(restaurants, r)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "error iterating result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(restaurants)
}

func GetDishesByRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantIDStr := mux.Vars(r)["id"]
	restaurantID, err := uuid.Parse(restaurantIDStr)
	if err != nil {
		http.Error(w, "Invalid restaurant ID", http.StatusBadRequest)
		return
	}

	type Dish struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
		IsAvailable bool      `json:"is_available"`
		CreatedAt   time.Time `json:"created_at"`
	}

	query := `
		SELECT id, name, description, price, is_available, created_at
		FROM menu_items
		WHERE restaurant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := database.Restro.Query(query, restaurantID)
	if err != nil {
		http.Error(w, "Failed to fetch dishes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dishes []Dish
	for rows.Next() {
		var d Dish
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.Price, &d.IsAvailable, &d.CreatedAt); err != nil {
			http.Error(w, "Failed to read dish data", http.StatusInternalServerError)
			return
		}
		dishes = append(dishes, d)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to iterate dishes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dishes)
}

func GetDistance(w http.ResponseWriter, r *http.Request) {
	// use haversine or OpenRouteServe (ORS); if using ORS register and get the API key which gives 2500requests/day
}

func ListResources(w http.ResponseWriter,r *http.Request) {
	resourceType := r.URL.Query().Get("type")
	if resourceType == "" {
		http.Error(w, "Missing resource type", http.StatusBadRequest)
		return
	}

	// Get user_id and roles from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roles, ok := r.Context().Value("roles").([]string)
	if !ok || len(roles) == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isAdmin := slices.Contains(roles, "admin")
	isSubAdmin := slices.Contains(roles, "subadmin")

	if !isAdmin && !isSubAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch resourceType {
	case "user":
		listUsers(w, userID, isAdmin)
	case "restaurant":
		listRestaurantsByCreator(w, userID, isAdmin)
	case "menu":
		listMenuItemsByCreator(w, userID, isAdmin)
	default:
		http.Error(w, "Invalid resource type", http.StatusBadRequest)
	}

}

func createUser(w http.ResponseWriter, r *http.Request, creatorID uuid.UUID) {
	type UserInput struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var input UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	var userID uuid.UUID
	err := database.Restro.QueryRow(`
		INSERT INTO users (name, email, password, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, input.Name, input.Email, string(hashedPassword), creatorID).Scan(&userID)
	if err != nil {
		http.Error(w, "User creation failed", http.StatusInternalServerError)
		return
	}

	role := "user"
	_, err = database.Restro.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, userID, role)
	if err != nil {
		http.Error(w, "Failed to assign role", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "User created",
		"user_id": userID.String(),
	})
}

func createRestaurant(w http.ResponseWriter, r *http.Request, creatorID uuid.UUID) {
	type Input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
	}

	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var restID uuid.UUID
	err := database.Restro.QueryRow(`
		INSERT INTO restaurants (name, owner_id, description, latitude, longitude, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, input.Name, creatorID, input.Description, input.Latitude, input.Longitude, creatorID).Scan(&restID)
	if err != nil {
		http.Error(w, "Failed to create restaurant", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message":      "Restaurant created",
		"restaurant_id": restID.String(),
	})
}

func createMenuItem(w http.ResponseWriter, r *http.Request, creatorID uuid.UUID) {
	type Input struct {
		RestaurantID uuid.UUID `json:"restaurant_id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		Price        float64   `json:"price"`
	}

	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var id uuid.UUID
	err := database.Restro.QueryRow(`
		INSERT INTO menu (restaurant_id, name, description, price, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, input.RestaurantID, input.Name, input.Description, input.Price, creatorID).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to create menu item", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message":     "Menu item created",
		"menu_item_id": id.String(),
	})
}

func listUsers(w http.ResponseWriter, userID uuid.UUID, isAdmin bool) {
	type User struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	}

	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = database.Restro.Query(`SELECT id, name, email FROM users WHERE archived_at IS NULL`)
	} else {
		rows, err = database.Restro.Query(`SELECT id, name, email FROM users WHERE created_by = $1 AND archived_at IS NULL`, userID)
	}

	if err != nil {
		http.Error(w, "Failed to query users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			http.Error(w, "Read error", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	json.NewEncoder(w).Encode(users)
}

func listRestaurantsByCreator(w http.ResponseWriter, userID uuid.UUID, isAdmin bool) {
	type Restaurant struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
	}

	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = database.Restro.Query(`SELECT id, name, description FROM restaurants`)
	} else {
		rows, err = database.Restro.Query(`SELECT id, name, description FROM restaurants WHERE created_by = $1`, userID)
	}

	if err != nil {
		http.Error(w, "Failed to query restaurants", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var restaurants []Restaurant
	for rows.Next() {
		var r Restaurant
		if err := rows.Scan(&r.ID, &r.Name, &r.Description); err != nil {
			http.Error(w, "Read error", http.StatusInternalServerError)
			return
		}
		restaurants = append(restaurants, r)
	}

	json.NewEncoder(w).Encode(restaurants)
}

func listMenuItemsByCreator(w http.ResponseWriter, userID uuid.UUID, isAdmin bool) {
	type MenuItem struct {
		ID           uuid.UUID `json:"id"`
		RestaurantID uuid.UUID `json:"restaurant_id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		Price        float64   `json:"price"`
	}

	var rows *sql.Rows
	var err error

	if isAdmin {
		rows, err = database.Restro.Query(`
			SELECT id, restaurant_id, name, description, price
			FROM menu_items
		`)
	} else {
		rows, err = database.Restro.Query(`
			SELECT id, restaurant_id, name, description, price
			FROM menu_items
			WHERE created_by = $1
		`, userID)
	}

	if err != nil {
		http.Error(w, "Failed to query menu items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []MenuItem
	for rows.Next() {
		var m MenuItem
		if err := rows.Scan(&m.ID, &m.RestaurantID, &m.Name, &m.Description, &m.Price); err != nil {
			http.Error(w, "Read error", http.StatusInternalServerError)
			return
		}
		items = append(items, m)
	}

	json.NewEncoder(w).Encode(items)
}
