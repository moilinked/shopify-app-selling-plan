package models

type Shop struct {
	CommonModel
	Name          string `gorm:"type:varchar(255);not null" json:"name"`
	AdminAPI      string `gorm:"type:varchar(500)" json:"adminApi"`
	OnlineShopURL string `gorm:"type:varchar(500)" json:"onlineShopUrl"`
	StorefrontAPI string `gorm:"type:varchar(500)" json:"storefrontApi"`
}
