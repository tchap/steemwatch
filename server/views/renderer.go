package views

import (
	"html/template"
	"io"
	"net/url"

	"github.com/labstack/echo"
)

type PageContext struct {
	CanonicalURL    *url.URL
	Environment     string
	UserId          string
	UserEmail       string
	UserDisplayName string
}

type Template struct {
	templates *template.Template
}

func NewRenderer(templateFilesPattern string) (*Template, error) {
	t, err := template.ParseGlob(templateFilesPattern)
	if err != nil {
		return nil, err
	}
	return &Template{t}, nil
}

func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
