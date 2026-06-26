package render

import "errors"

// ErrTemplateRendererRequired indicates that a page with a layout cannot be
// rendered because no template renderer was provided.
var ErrTemplateRendererRequired = errors.New("template renderer is required")
