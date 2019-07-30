package helpers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/bitly/go-simplejson"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// Convert convierte todos los archivos YAML que se encuentren en source/**/*.yaml
// a archivos JSON ubicados en target/ conservando el path relativo.
func Convert(source, target string) error {
	ss, err := sources(source)
	if err != nil {
		return errors.Wrapf(err, "looking for sources in %s", source)
	}
	for _, s := range ss {
		t := filepath.Join(target, chext(s, ".json"))
		log.Printf("Transforming %s to %s", s, t)
		err := transform(s, t)
		if err != nil {
			return errors.Wrapf(err, "transforming %s", s)
		}
	}
	return nil
}

func chext(s string, ext string) string {
	return strings.TrimSuffix(filepath.Base(s), filepath.Ext(s)) + ext
}

func sources(src string) ([]string, error) {
	res := []string{}
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		m, errr := filepath.Match("*.yaml", filepath.Base(path))
		if errr != nil {
			return errr
		}
		if m {
			res = append(res, path)
		}
		return nil
	})
	return res, err
}

func transform(src, tgt string) error {
	bs, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	bs, err = FromYAML(bs, src, tgt, true)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tgt, bs, 0664)
}

func FromYAML(bs []byte, src, tgt string, pretty bool) ([]byte, error) {
	bs, err := yaml.YAMLToJSON(bs)
	if err != nil {
		return nil, err
	}
	bs, err = clean(bs)
	if err != nil {
		return nil, err
	}
	bs, err = patch(bs, src, tgt)
	if err != nil {
		return nil, err
	}
	if pretty {
		bs, err = Pretty(bs)
		if err != nil {
			return nil, err
		}
	}
	return bs, nil
}

const p = `
{
	"$meta": {
		{{ if .source }}"source": "{{ .source }}",{{ end }}
		"comment": "SCHEMA GENERADO AUTOMÁTICAMENTE (NO MODIFICAR)"
	}
}`

var tpl = template.Must(template.New("patch").Parse(p))

func patch(bs []byte, src, tgt string) ([]byte, error) {
	b := bytes.Buffer{}
	err := tpl.Execute(&b, map[string]string{"source": src, "target": tgt})
	if err != nil {
		return nil, err
	}
	return jsonpatch.MergePatch(bs, b.Bytes())
}

func clean(bs []byte) ([]byte, error) {
	v, err := simplejson.NewJson(bs)
	if err != nil {
		return nil, err
	}
	filter(v)
	return v.Encode()
}

func filter(v *simplejson.Json) {
	m, err := v.Map()
	if err != nil {
		return
	}
	for k := range m {
		if strings.HasPrefix(k, "x-") {
			v.Del(k)
		} else {
			filter(v.Get(k))
		}
	}
}

func Pretty(bs []byte) ([]byte, error) {
	b := bytes.Buffer{}
	err := json.Indent(&b, bs, "", "  ")
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
