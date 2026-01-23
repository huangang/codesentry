package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Project struct {
	ID             uint   `gorm:"primaryKey"`
	Name           string `gorm:"size:200;not null"`
	URL            string `gorm:"size:500;not null"`
	IgnorePatterns string `gorm:"size:2000"`
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

	// 新的 ignore_patterns 值
	newIgnorePatterns := "etc/config.yaml,etc/.env,etc/config.yml"

	// 查询更新前的数据（只显示前10个示例）
	var sampleProjects []Project
	if err := db.Where("deleted_at IS NULL").Limit(10).Find(&sampleProjects).Error; err != nil {
		log.Fatalf("Failed to query projects: %v", err)
	}

	fmt.Println("Sample projects before update (showing first 10):")
	fmt.Printf("%-5s %-40s %-60s\n", "ID", "Name", "IgnorePatterns")
	fmt.Println("----------------------------------------------------------------------------------------------------------------")
	for _, p := range sampleProjects {
		patterns := p.IgnorePatterns
		if len(patterns) > 60 {
			patterns = patterns[:57] + "..."
		}
		fmt.Printf("%-5d %-40s %-60s\n", p.ID, p.Name, patterns)
	}
	fmt.Println("")

	// 统计总数
	var totalCount int64
	db.Model(&Project{}).Where("deleted_at IS NULL").Count(&totalCount)
	fmt.Printf("Total projects to update: %d\n", totalCount)
	fmt.Println("")

	// 执行更新
	result := db.Model(&Project{}).
		Where("deleted_at IS NULL").
		Update("ignore_patterns", newIgnorePatterns)

	if result.Error != nil {
		log.Fatalf("Failed to update projects: %v", result.Error)
	}

	fmt.Printf("✅ Successfully updated %d projects!\n", result.RowsAffected)
	fmt.Println("")

	// 查询更新后的数据（显示前10个示例）
	var updatedSampleProjects []Project
	if err := db.Where("deleted_at IS NULL").Limit(10).Find(&updatedSampleProjects).Error; err != nil {
		log.Fatalf("Failed to query updated projects: %v", err)
	}

	fmt.Println("Sample projects after update (showing first 10):")
	fmt.Printf("%-5s %-40s %-60s\n", "ID", "Name", "IgnorePatterns")
	fmt.Println("----------------------------------------------------------------------------------------------------------------")
	for _, p := range updatedSampleProjects {
		fmt.Printf("%-5d %-40s %-60s\n", p.ID, p.Name, p.IgnorePatterns)
	}

	fmt.Println("")
	fmt.Printf("✅ All projects now have IgnorePatterns = '%s'\n", newIgnorePatterns)
}
