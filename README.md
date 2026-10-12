# cyber-blog

A modern, sleek personal blog backend built with Go, Gin, GORM, PostgreSQL, and Cloudflare R2.
Designed with Web3 aesthetic sensibilities and cloud-native resilience.

## Features
- **Authentication**: JWT Bearer token authentication & password encryption with bcrypt
- **Post Management**: CRUD operations with pagination & author-only modifications
- **Comments**: Reader discussion & interaction under articles
- **Media Storage**: Cloudflare R2 object storage integration (S3 compatible)
- **Database**: PostgreSQL 18 with GORM auto-migration and SQLite fallback for local testing
