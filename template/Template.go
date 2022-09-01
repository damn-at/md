package template

import (
	"bytes"
	"fmt"
	"github.com/yuin/goldmark"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ParseGlob(debug bool, md goldmark.Markdown, t *template.Template, glob string) (*template.Template, error) {
	if md == nil {
		return nil, fmt.Errorf("MD was nil")
	}
	if t == nil {
		return nil, fmt.Errorf("template.Template was nil")
	}

	filenames, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("md/template: pattern matches no files: %#q", glob)
	}

	return parseFiles(debug, md, t, "", readFileOS, filenames...)
}

func ParseString(debug bool, md goldmark.Markdown, t *template.Template, templateName string, markdown string) (*template.Template, error) {
	if md == nil {
		return nil, fmt.Errorf("MD was nil")
	}
	if t == nil {
		return nil, fmt.Errorf("template.Template was nil")
	}

	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		err = fmt.Errorf(`
failed to convert Markdown to HTML: %v

Markdown:
%v

`, err, markdown)
		panic(err)
	}
	var re = regexp.MustCompile(`(?m){{\s*template\s*&ldquo;(?P<Name>.*)&rdquo;\s*(?P<Parameter>.*)\s*}}`)
	var substitution = "{{ template \"${Name}\" ${Parameter} }}"
	mdTemplate := re.ReplaceAllString(buf.String(), substitution)
  mdTemplate = strings.ReplaceAll(mdTemplate, "&ldquo;", "\"")
	mdTemplate = strings.ReplaceAll(mdTemplate, "&rdquo;", "\"")

	var html bytes.Buffer
	html.WriteString(fmt.Sprintf(`{{ define "%s" }}`, templateName))
	html.WriteString("\n")
	html.WriteString(mdTemplate)
	html.WriteString(`{{ end }}`)

	// First template becomes return value if not already defined,
	// and we use that one for subsequent New calls to associate
	// all the templates together. Also, if this file has the same name
	// as t, this file becomes the contents of t, so
	//  t, err := New(name).Funcs(xxx).ParseFiles(name)
	// works. Otherwise we create a new template associated with t.
	var tmpl *template.Template
	if t == nil {
		t = template.New(templateName)
	}
	if templateName == t.Name() {
		tmpl = t
	} else {
		tmpl = t.New(templateName)
	}
	var err error
	t, err = tmpl.Parse(html.String())
	if err != nil {
		err = fmt.Errorf("%s\n Template:\n%s", err.Error(), html.String())
		return nil, err
	}
	return t, nil
}

func Parse(debug bool, md goldmark.Markdown, t *template.Template, templateName string, file string) (*template.Template, error) {
	if md == nil {
		return nil, fmt.Errorf("MD was nil")
	}
	if t == nil {
		return nil, fmt.Errorf("template.Template was nil")
	}

	return parseFiles(debug, md, t, templateName, readFileOS, file)
}

// parseFiles is the helper for the method and function. If the argument
// template is nil, it is created from the first file.
func parseFiles(debug bool, md goldmark.Markdown, t *template.Template, forceTemplateName string, readFile func(string) (string, []byte, error), filenames ...string) (*template.Template, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("md/template: no files named in call to ParseFiles")
	}
	for _, filename := range filenames {
		name, b, err := readFile(filename)
		if err != nil {
			return nil, err
		}
		templateName := strings.ReplaceAll(name, ".go.md", "")
		var buf bytes.Buffer
		if err := md.Convert(b, &buf); err != nil {
			err = fmt.Errorf(`
failed to convert Markdown to HTML: %v

Markdown:
%v

`, err, string(b))
			panic(err)
		}
		var re = regexp.MustCompile(`(?m){{\s*template\s*&ldquo;(?P<Name>.*)&rdquo;\s*(?P<Parameter>.*)\s*}}`)
		var substitution = "{{ template \"${Name}\" ${Parameter} }}"
		mdTemplate := re.ReplaceAllString(buf.String(), substitution)
		mdTemplate = strings.ReplaceAll(mdTemplate, "&ldquo;", "\"")
		mdTemplate = strings.ReplaceAll(mdTemplate, "&rdquo;", "\"")

		var html bytes.Buffer
		if forceTemplateName != "" {
			templateName = forceTemplateName
		}
		html.WriteString(fmt.Sprintf(`{{ define "%s" }}`, templateName))
		html.WriteString("\n")
		html.WriteString(mdTemplate)
		html.WriteString(`{{ end }}`)

		// First template becomes return value if not already defined,
		// and we use that one for subsequent New calls to associate
		// all the templates together. Also, if this file has the same name
		// as t, this file becomes the contents of t, so
		//  t, err := New(name).Funcs(xxx).ParseFiles(name)
		// works. Otherwise we create a new template associated with t.
		var tmpl *template.Template
		if t == nil {
			t = template.New(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(templateName)
		}
		t, err = tmpl.Parse(html.String())
		if err != nil {
			err = fmt.Errorf("%s\n Template:\n%s", err.Error(), html.String())
			return nil, err
		}
	}
	return t, nil
}

func readFileOS(file string) (name string, b []byte, err error) {
	name = filepath.Base(file)
	b, err = os.ReadFile(file)
	return
}
