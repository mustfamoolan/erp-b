package inventory

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================
// ITEM MASTER DATA — Roadmap §25
// Category → Item → Variant → SKU + Barcode
// ============================================================

// ItemCategory groups items hierarchically.
type ItemCategory struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name     string     `gorm:"type:varchar(150);not null"`
	ParentID *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ItemCategory) TableName() string { return "item_categories" }

// UnitOfMeasure defines how items are counted/measured.
type UnitOfMeasure struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code      string    `gorm:"type:varchar(20);not null;uniqueIndex"` // e.g. "KG", "PCS", "M"
	Name      string    `gorm:"type:varchar(100);not null"`
	CreatedAt time.Time
}

func (UnitOfMeasure) TableName() string { return "units_of_measure" }

// Item is the base material/product record.
type Item struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Code        string    `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name        string    `gorm:"type:varchar(200);not null"`
	Description string    `gorm:"type:text"`
	BaseUnitID  uuid.UUID `gorm:"type:uuid;not null"` // base unit of measure
	IsActive    bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Variants []ItemVariant `gorm:"foreignKey:ItemID"`
}

func (Item) TableName() string { return "items" }

// ItemVariant is a specific variant of an item with its own SKU and barcode.
// Roadmap §25 example: Iron Sheet 3mm 120cm 240cm
type ItemVariant struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ItemID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Name      string    `gorm:"type:varchar(200);not null"`
	SKU       string    `gorm:"type:varchar(100);not null;uniqueIndex"` // business identifier — §26
	Barcode   string    `gorm:"type:varchar(100);uniqueIndex"`           // machine-readable — §26
	UnitID    uuid.UUID `gorm:"type:uuid;not null"`
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Attributes []ItemVariantAttribute `gorm:"foreignKey:VariantID"`
}

func (ItemVariant) TableName() string { return "item_variants" }

// ItemVariantAttribute stores key-value attributes (Thickness=3mm, Width=120cm).
type ItemVariantAttribute struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VariantID uuid.UUID `gorm:"type:uuid;not null;index"`
	AttrKey   string    `gorm:"type:varchar(100);not null"`
	AttrValue string    `gorm:"type:varchar(200);not null"`
}

func (ItemVariantAttribute) TableName() string { return "item_variant_attributes" }

// ============================================================
// WAREHOUSE — Roadmap §27
// Warehouses are owned by scopes. Rule 5, Rule 6.
// ============================================================

// WarehouseType classifies the warehouse.
type WarehouseType string

const (
	WarehouseTypeRawMaterial    WarehouseType = "RAW_MATERIAL"
	WarehouseTypeFinishedGoods  WarehouseType = "FINISHED_GOODS"
	WarehouseTypeConsumables    WarehouseType = "CONSUMABLES"
	WarehouseTypeGeneral        WarehouseType = "GENERAL"
)

// Warehouse is a physical or logical storage location owned by a scope.
type Warehouse struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ScopeID   uuid.UUID     `gorm:"type:uuid;not null;index"` // who owns this warehouse — Rule 5
	Code      string        `gorm:"type:varchar(50);not null;uniqueIndex"`
	Name      string        `gorm:"type:varchar(150);not null"`
	Type      WarehouseType `gorm:"type:varchar(30);not null;default:'GENERAL'"`
	IsActive  bool          `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Warehouse) TableName() string { return "warehouses" }

// ============================================================
// STOCK MOVEMENT — Roadmap §28
// Stock quantity is reconstructable from movements.
// Rule 12: Every inventory movement must have a traceable stock movement.
// Rule 16: Inventory corrections use adjustment movements.
// ============================================================

// StockMovementType defines the reason for a stock movement.
type StockMovementType string

const (
	StockMovementOpeningBalance  StockMovementType = "OPENING_BALANCE"
	StockMovementPurchaseReceipt StockMovementType = "PURCHASE_RECEIPT"
	StockMovementTransferIn      StockMovementType = "TRANSFER_IN"
	StockMovementTransferOut     StockMovementType = "TRANSFER_OUT"
	StockMovementConsumption     StockMovementType = "CONSUMPTION"
	StockMovementProduction      StockMovementType = "PRODUCTION"
	StockMovementAdjustment      StockMovementType = "ADJUSTMENT"
	StockMovementReturn          StockMovementType = "RETURN"
	StockMovementIssue           StockMovementType = "ISSUE"
	StockMovementReceipt         StockMovementType = "RECEIPT"
)

// StockMovement is an immutable record of every inventory change.
// Stock quantity is ALWAYS reconstructed from movements — never from a stored field.
type StockMovement struct {
	ID            uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WarehouseID   uuid.UUID         `gorm:"type:uuid;not null;index"`
	VariantID     uuid.UUID         `gorm:"type:uuid;not null;index"` // item variant
	Quantity      decimal.Decimal   `gorm:"type:numeric(18,4);not null"`   // positive = in, negative = out
	UnitID        uuid.UUID         `gorm:"type:uuid;not null"`
	MovementType  StockMovementType `gorm:"type:varchar(30);not null"`
	ReferenceType string            `gorm:"type:varchar(50)"`  // polymorphic source
	ReferenceID   *uuid.UUID        `gorm:"type:uuid;index"`
	PerformedBy   uuid.UUID         `gorm:"type:uuid;not null"`
	Notes         string            `gorm:"type:text"`
	CreatedAt     time.Time
}

func (StockMovement) TableName() string { return "stock_movements" }
