package model

// Image 对应于数据库中的 'images' 表
type Image struct {
	ID        int    `json:"id" gorm:"column:id;primaryKey"`
	Name      string `json:"name" gorm:"column:name"`
	URL       string `json:"url" gorm:"column:url"`
	LocalPath string `json:"local_path" gorm:"column:local_path"`
	IsLocal   int    `json:"is_local" gorm:"column:is_local"`
	IsOss     int    `json:"is_oss" gorm:"column:is_oss"`
	Status    string `json:"status" gorm:"column:status"`
}

func (Image) TableName() string {
	return "images"
}

type QueryImageResponse struct {
	Images []Image
	Total  int64
}
