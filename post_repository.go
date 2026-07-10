package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/dromara/carbon/v2"
)

var (
	ErrDuplicatePostTitle = errors.New("Title already exists")
	ErrDuplicateVote      = errors.New("Duplicate vote found")
)

type Post struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	UserID       int       `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UserName     string    `json:"user_name"`
	CommentCount int       `json:"comment_count"`
	VoteCount    int       `json:"vote_count"`
	TotalRecords int       `json:"total_records"`
}

type Comment struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

type Filter struct {
	Page     int    `json:"page"`      // Which page of results are we on? (e.g. Page 1, Page 2)
	PageSize int    `json:"page_size"` // How many items do we want per page? (e.g. 10, 20, 50)
	OrderBy  string `json:"order_by"`  // How should we sort them? (e.g. "newest", "top_voted")
	Query    string `json:"query"`     // Search keyword (e.g. searching for "golang")
}

type Vote struct {
	UserID    int       `json:"user_id"`
	PostID    int       `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (f *Filter) Validate() error {
	if f.PageSize <= 0 || f.PageSize >= 100 {
		return errors.New("Invalid page size")
	}
	return nil
}

type Metadata struct {
	CurrentPage  int `json:"current_page"`  // The page the user is currently looking at
	PageSize     int `json:"page_size"`     // How many items are on this page
	FirstPage    int `json:"first_page"`    // Always 1, used for the "Go to First" button
	NextPage     int `json:"next_page"`     // The page number for the "Next" button
	PrevPage     int `json:"prev_page"`     // The page number for the "Prev" button
	LastPage     int `json:"last_page"`     // The total number of pages available
	TotalRecords int `json:"total_records"` // The absolute total number of posts in the DB matching the query
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	meta := Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}

	if meta.CurrentPage > 1 {
		meta.PrevPage = meta.CurrentPage - 1
	} else {
		meta.PrevPage = 0
	}
	if meta.CurrentPage < meta.LastPage {
		meta.NextPage = meta.CurrentPage + 1
	} else {
		meta.NextPage = 0
	}
	return meta
}

type PostRepository interface {
	CreatePost(title, url string, userID int) (int, error)
	AddComment(userID, postID int, body string) (int, error)
	AddVote(userID, postID int) error
	GetAll(filter Filter) ([]Post, Metadata, error)
	GetByID(id int) (*Post, error)
	GetComments(postID int) ([]Comment, error)
}

type SQLPostRepository struct {
	db *sql.DB
}

func NewSQLPostRepository(db *sql.DB) PostRepository {
	return &SQLPostRepository{db: db}
}

func (r *SQLPostRepository) CreatePost(title, url string, userID int) (int, error) {
	// Query
	query := `INSERT INTO posts (title, url, user_id) VALUES (?, ?, ?)`

	// Execute
	result, err := r.db.Exec(query, title, url, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: posts.title") {
			return 0, ErrDuplicatePostTitle
		}
		return 0, err
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(lastInsertID), nil
}

func (r *SQLPostRepository) AddComment(userID, postID int, body string) (int, error) {
	// Query
	query := `INSERT INTO comments (user_id, post_id, body) VALUES (?, ?, ?)`

	// Execute
	result, err := r.db.Exec(query, userID, postID, body)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(lastInsertID), nil
}

func (r *SQLPostRepository) AddVote(userID, postID int) error {
	// Query
	query := `INSERT INTO votes (user_id, post_id) VALUES (?, ?)`

	// Execute
	_, err := r.db.Exec(query, userID, postID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "PRIMARY KEY constraint failed") {
			return ErrDuplicateVote
		}
		return err
	}

	return nil
}

func (r *SQLPostRepository) GetByID(id int) (*Post, error) {
	query := `
	SELECT p.id, p.title, p.url, p.user_id, p.created_at,
	u.name AS user_name,
	COUNT(DISTINCT c.id) AS comment_count,
	COUNT(DISTINCT v.post_id) AS vote_count
	FROM posts p
	INNER JOIN users u ON u.id = p.user_id
	LEFT JOIN comments c ON p.id = c.post_id
	LEFT JOIN votes v ON p.id = v.post_id
	WHERE p.id = ?
	GROUP BY p.id, p.title, p.url, p.user_id, p.created_at, u.name
	LIMIT 1
	`

	row := r.db.QueryRow(query, id)
	var post Post
	err := row.Scan(&post.ID, &post.Title, &post.URL, &post.UserID, &post.CreatedAt, &post.UserName, &post.CommentCount, &post.VoteCount)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *SQLPostRepository) GetAll(filter Filter) ([]Post, Metadata, error) {
	// 1. Sanitize input defaults
	if err := filter.Validate(); err != nil {
		return nil, Metadata{}, err
	}

	// 2. Optimized Query (No Cartesian explosion, No GROUP BY)
	query := `
		SELECT p.id, p.title, p.url, p.user_id, p.created_at,
		u.name AS user_name,
		(SELECT COUNT(*) FROM comments WHERE post_id = p.id) AS comment_count,
		(SELECT COUNT(*) FROM votes WHERE post_id = p.id) AS vote_count,
		COUNT(*) OVER() AS total_records
		FROM posts p
		INNER JOIN users u ON u.id = p.user_id
	`

	var args []interface{}
	var queryBuilder strings.Builder
	queryBuilder.WriteString(query)

	if filter.Query != "" {
		queryBuilder.WriteString(" WHERE LOWER(p.title) LIKE ?")
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
	}

	var orderClause string
	switch filter.OrderBy {
	case "popular":
		orderClause = " ORDER BY vote_count DESC, p.created_at DESC"
	case "comment_count":
		orderClause = " ORDER BY comment_count DESC"
	case "vote_count":
		orderClause = " ORDER BY vote_count DESC"
	default:
		orderClause = " ORDER BY p.created_at DESC"
	}
	queryBuilder.WriteString(orderClause)

	limit := filter.PageSize
	offset := (filter.Page - 1) * filter.PageSize
	queryBuilder.WriteString(" LIMIT ? OFFSET ?")
	args = append(args, limit, offset)

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	// 3. Pre-allocate slice capacity to optimize memory allocation
	posts := make([]Post, 0, limit)

	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID, &post.Title, &post.URL, &post.UserID, &post.CreatedAt,
			&post.UserName, &post.CommentCount, &post.VoteCount, &post.TotalRecords,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	var meta Metadata
	if len(posts) > 0 {
		meta = calculateMetadata(posts[0].TotalRecords, filter.Page, filter.PageSize)
	}
	return posts, meta, nil
}

func (r *SQLPostRepository) GetComments(postID int) ([]Comment, error) {
	query := `
	SELECT c.id, c.body, c.user_id, c.post_id, c.created_at, 
	u.name as user_name 
	FROM comments c 
	INNER JOIN users u ON c.user_id = u.id
	WHERE c.post_id = ?
	ORDER BY c.created_at DESC
	`

	rows, err := r.db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []Comment{}

	for rows.Next(){
		var comment Comment

		err := rows.Scan(&comment.ID, &comment.Body, &comment.UserID, &comment.PostID, &comment.CreatedAt, &comment.UserName)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (p *Post) GetVoteCountsHuman() string {
	if p.VoteCount > 1 {
		return fmt.Sprintf("%d votes", p.VoteCount)
	}

	return fmt.Sprintf("%d vote", p.VoteCount)
}

func (p *Post) GetCommentCountsHuman() string {
	if p.CommentCount > 1 {
		return fmt.Sprintf("%d comments", p.CommentCount)
	}

	return fmt.Sprintf("%d comment", p.CommentCount)
}

func (p *Post) CreatedAtHuman() string {
	return carbon.NewCarbon(p.CreatedAt).DiffForHumans()
}

func (p *Post) Host() string {
	parsedURL, err := url.Parse(p.URL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsedURL.Hostname(), "www.")
}