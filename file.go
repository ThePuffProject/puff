package puff

import (
	"fmt"
	"mime/multipart"

	"github.com/ThePuffProject/puff/openapi"
)

// File represents a file in Puff. WARNING: the Name field has not been sanitized.
type File struct {
	multipart.File
	Name string
	Size int64
}

func getFileParam(c *Context, p *openapi.Parameter) (*File, error) {
	file, header, err := c.GetFormFile(p.Name)
	if err != nil {
		return nil, fmt.Errorf("get file error: %v", err)
	}
	// FIXME: validate MIME
	return &File{
		File: file,
		Name: header.Filename,
		Size: header.Size,
	}, nil
}
