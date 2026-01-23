package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Project struct {
	ID       uint    `gorm:"primaryKey"`
	Name     string  `gorm:"size:200;not null"`
	URL      string  `gorm:"size:500;not null"`
	MinScore float64 `gorm:"default:0"`
}

func (Project) TableName() string {
	return "projects"
}

func main() {
	// 数据库连接
	dsn := "downtown:downtown#2013@tcp(10.11.15.44:3306)/codesentry?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Connected to database successfully!")
	fmt.Println("")

	// 查询更新前的数据
	var projects []Project
	if err := db.Where("deleted_at IS NULL").Find(&projects).Error; err != nil {
		log.Fatalf("Failed to query projects: %v", err)
	}

	fmt.Println("Projects before update:")
	fmt.Printf("%-5s %-30s %-50s %-10s\n", "ID", "Name", "URL", "MinScore")
	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
	for _, p := range projects {
		fmt.Printf("%-5d %-30s %-50s %-10.2f\n", p.ID, p.Name, p.URL, p.MinScore)
	}
	fmt.Println("")

	// 执行更新
	result := db.Model(&Project{}).Where("deleted_at IS NULL").Update("min_score", 75)
	if result.Error != nil {
		log.Fatalf("Failed to update projects: %v", result.Error)
	}

	fmt.Printf("✅ Successfully updated %d projects!\n", result.RowsAffected)
	fmt.Println("")

	// 查询更新后的数据
	var updatedProjects []Project
	if err := db.Where("deleted_at IS NULL").Find(&updatedProjects).Error; err != nil {
		log.Fatalf("Failed to query updated projects: %v", err)
	}

	fmt.Println("Projects after update:")
	fmt.Printf("%-5s %-30s %-50s %-10s\n", "ID", "Name", "URL", "MinScore")
	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
	for _, p := range updatedProjects {
		fmt.Printf("%-5d %-30s %-50s %-10.2f\n", p.ID, p.Name, p.URL, p.MinScore)
	}

	fmt.Println("")
	fmt.Printf("✅ All projects now have MinScore = 75\n")
}
