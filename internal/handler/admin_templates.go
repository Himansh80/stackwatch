// Tier 5 — Templates (C11).
//
//   GET /api/v1/containers/templates
//
// Returns a hardcoded list of common compose templates. No auth
// required — templates are public knowledge.
package handler

import (
	"github.com/gin-gonic/gin"
)

// template is one compose template.
type template struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ComposeYAML string            `json:"compose_yaml"`
	Tags        []string          `json:"tags"`
}

// templates is the built-in template library.
var templates = []template{
	{
		Name:        "nginx",
		Description: "Reverse proxy + static site on port 80",
		Tags:        []string{"web", "proxy"},
		ComposeYAML: `services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    restart: unless-stopped
`,
	},
	{
		Name:        "postgres",
		Description: "PostgreSQL 16 database on port 5432",
		Tags:        []string{"database", "sql"},
		ComposeYAML: `services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: change-me-in-production
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    restart: unless-stopped
volumes:
  pgdata:
`,
	},
	{
		Name:        "redis",
		Description: "Redis 7 cache on port 6379",
		Tags:        []string{"cache", "kv"},
		ComposeYAML: `services:
  cache:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    restart: unless-stopped
`,
	},
	{
		Name:        "ghost",
		Description: "Ghost blog with MySQL on port 2368",
		Tags:        []string{"blog", "cms"},
		ComposeYAML: `services:
  ghost:
    image: ghost:5-alpine
    environment:
      database__client: mysql
      database__connection__host: db
      database__connection__user: ghost
      database__connection__password: change-me
      database__connection__database: ghost
    depends_on:
      - db
    ports:
      - "2368:2368"
    restart: unless-stopped
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: change-me
      MYSQL_DATABASE: ghost
      MYSQL_USER: ghost
      MYSQL_PASSWORD: change-me
    volumes:
      - mysqldata:/var/lib/mysql
    restart: unless-stopped
volumes:
  mysqldata:
`,
	},
	{
		Name:        "nextcloud",
		Description: "Nextcloud self-hosted file share on port 8080",
		Tags:        []string{"storage", "files"},
		ComposeYAML: `services:
  nextcloud:
    image: nextcloud:28-apache
    depends_on:
      - db
    ports:
      - "8080:80"
    volumes:
      - ncdata:/var/www/html
    restart: unless-stopped
  db:
    image: mariadb:11
    environment:
      MYSQL_ROOT_PASSWORD: change-me
      MYSQL_DATABASE: nextcloud
      MYSQL_USER: nextcloud
      MYSQL_PASSWORD: change-me
    volumes:
      - ncdb:/var/lib/mysql
    restart: unless-stopped
volumes:
  ncdata:
  ncdb:
`,
	},
	{
		Name:        "prometheus-grafana",
		Description: "Prometheus + Grafana monitoring stack",
		Tags:        []string{"monitoring", "metrics"},
		ComposeYAML: `services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    restart: unless-stopped
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: change-me
    depends_on:
      - prometheus
    restart: unless-stopped
`,
	},
}

// ListTemplates returns all built-in templates. No auth needed.
func (h *ContainerHandler) ListTemplates(c *gin.Context) {
	c.JSON(200, gin.H{
		"templates": templates,
		"total":     len(templates),
	})
}
