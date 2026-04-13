package discovery

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// YUMParser streams packages from a gzip-compressed YUM primary.xml file.
type YUMParser struct {
	OsFamily   string
	OsDistro   string
	SourceRepo string
}

// Parse reads a gzip-compressed primary.xml and yields DiscoveredPatch values.
func (p *YUMParser) Parse(ctx context.Context, r io.Reader) func(yield func(DiscoveredPatch, error) bool) {
	return func(yield func(DiscoveredPatch, error) bool) {
		gz, err := gzip.NewReader(r)
		if err != nil {
			yield(DiscoveredPatch{}, fmt.Errorf("yum parser: open gzip: %w", err))
			return
		}
		defer gz.Close()

		decoder := xml.NewDecoder(gz)
		for {
			if ctx.Err() != nil {
				return
			}
			tok, err := decoder.Token()
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(DiscoveredPatch{}, fmt.Errorf("yum parser: read token: %w", err))
				return
			}
			se, ok := tok.(xml.StartElement)
			if !ok || se.Name.Local != "package" {
				continue
			}
			var pkg yumPackage
			if err := decoder.DecodeElement(&pkg, &se); err != nil {
				yield(DiscoveredPatch{}, fmt.Errorf("yum parser: decode package: %w", err))
				return
			}
			if !yield(p.toPatch(pkg), nil) {
				return
			}
		}
	}
}

type yumPackage struct {
	Name     string      `xml:"name"`
	Arch     string      `xml:"arch"`
	Version  yumVersion  `xml:"version"`
	Checksum yumChecksum `xml:"checksum"`
	Summary  string      `xml:"summary"`
	Desc     string      `xml:"description"`
	Size     yumSize     `xml:"size"`
	Location yumLocation `xml:"location"`
}

type yumVersion struct {
	Epoch string `xml:"epoch,attr"`
	Ver   string `xml:"ver,attr"`
	Rel   string `xml:"rel,attr"`
}

type yumChecksum struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type yumSize struct {
	Package string `xml:"package,attr"`
}

type yumLocation struct {
	Href string `xml:"href,attr"`
}

func (p *YUMParser) toPatch(pkg yumPackage) DiscoveredPatch {
	size, _ := strconv.ParseInt(pkg.Size.Package, 10, 64)
	version := pkg.Version.Ver
	if pkg.Version.Rel != "" {
		version += "-" + pkg.Version.Rel
	}
	if pkg.Version.Epoch != "" && pkg.Version.Epoch != "0" {
		version = pkg.Version.Epoch + ":" + version
	}
	return DiscoveredPatch{
		Name:        pkg.Name,
		Version:     version,
		Arch:        pkg.Arch,
		OsFamily:    p.OsFamily,
		OsDistro:    p.OsDistro,
		Summary:     pkg.Summary,
		Description: pkg.Desc,
		Filename:    pkg.Location.Href,
		Size:        size,
		Checksum:    pkg.Checksum.Value,
		SourceRepo:  p.SourceRepo,
	}
}
