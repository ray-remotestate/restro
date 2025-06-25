package models

import(
	"time"

	"github.com/google/uuid"
)

type Restaurant struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	OwnerID     uuid.UUID `db:"owner_id" json:"owner_id"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type MenuItem struct {
	ID           uuid.UUID `db:"id" json:"id"`
	RestaurantID uuid.UUID `db:"restaurant_id" json:"restaurant_id"`
	Name         string    `db:"name" json:"name"`
	Description  string    `db:"description" json:"description"`
	Price        float64   `db:"price" json:"price"`
	IsAvailable  bool      `db:"is_available" json:"is_available"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Order struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	RestaurantID uuid.UUID `db:"restaurant_id" json:"restaurant_id"`
	Status     string    `db:"status" json:"status"` // pending, paid, cancelled
	Total      float64   `db:"total" json:"total"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type OrderItem struct {
	ID        uuid.UUID `db:"id" json:"id"`
	OrderID   uuid.UUID `db:"order_id" json:"order_id"`
	MenuItemID uuid.UUID `db:"menu_item_id" json:"menu_item_id"`
	Quantity  int       `db:"quantity" json:"quantity"`
	Price     float64   `db:"price" json:"price"`
}
