package render

import (
	"fmt"
	"io"
	"log"

	"github.com/CloudyKit/jet/v6"
	"github.com/labstack/echo/v4"
)

type Renderer struct {
	views *jet.Set
}

type TemplateData struct {
	IsAuthenticated bool
	IntMap          map[string]int
	StringMap       map[string]string
	FloatMap        map[string]float32
	Data            map[string]any
}

func NewRenderer(root string, development bool) *Renderer {
	loader := jet.NewOSFileSystemLoader(root)

	opts := []jet.Option{}
	if development {
		opts = append(opts, jet.InDevelopmentMode())
	}

	return &Renderer{
		views: jet.NewSet(loader, opts...),
	}
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, err := r.views.GetTemplate(fmt.Sprintf("%s.jet", name))
	if err != nil {
		log.Println("Template load error:", err)
		return err
	}

	td := toTemplateData(data)

	// Build Jet VarMap
	vars := make(jet.VarMap)

	// Promote TemplateData fields into Jet globals
	vars.Set("StringMap", td.StringMap)
	vars.Set("IntMap", td.IntMap)
	vars.Set("FloatMap", td.FloatMap)
	vars.Set("Data", td.Data)
	vars.Set("IsAuthenticated", td.IsAuthenticated)

	// PASS td AS THE EXECUTION CONTEXT (3rd parameter)
	return tmpl.Execute(w, vars, td)
}

func toTemplateData(data interface{}) *TemplateData {
	switch v := data.(type) {
	case *TemplateData:
		return v
	case TemplateData:
		return &v
	case map[string]any:
		return &TemplateData{Data: v}
	case nil:
		return &TemplateData{}
	default:
		return &TemplateData{
			Data: map[string]any{
				"data": v,
			},
		}
	}
}
