package epub

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

var (
	mtXHTML = "application/xhtml+xml"
	mtNCX   = "application/x-dtbncx+xml"
	mtOPF   = "application/opf+xml"
)

func (e *Book) Write(filename string) error {
	// write cover.xhtml
	err := e.execTemplate("cover.xhtml", "text/cover.xhtml", mtXHTML)
	if err != nil {
		return err
	}

	// write sections
	for _, section := range e.sections[0] {
		err = e.buildSection(section, "introduction")
		if err != nil {
			return err
		}
	}
	for _, section := range e.sections[1] {
		err = e.buildSection(section, "chapter")
		if err != nil {
			return err
		}
	}
	for _, section := range e.sections[2] {
		err = e.buildSection(section, "postscript")
		if err != nil {
			return err
		}
	}

	// write text/toc.html
	err = e.execTemplate("toc.xhtml", "text/toc.xhtml", mtXHTML)
	if err != nil {
		return err
	}

	// write toc.ncx
	if err := e.execTemplate("toc.ncx", "toc.ncx", mtNCX); err != nil {
		return err
	}

	// write book.opf
	if err := e.execTemplate("book.opf", "book.opf", mtOPF); err != nil {
		return err
	}

	if err := e.file.Close(); err != nil {
		return errors.Wrap(
			err,
			"Close zip file",
		)
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, e.buf)
	if err != nil {
		return err
	}
	return f.Close()

}

func (e *Book) execTemplate(filename, zipName, mediaType string) error {
	tt, err := CompileTemplate(filename)
	if err != nil {
		return err
	}
	e.args.Files = append(e.args.Files, bookFile{
		ID:        filepath.Base(zipName),
		Path:      zipName,
		MediaType: mediaType,
	})
	buf, _ := e.file.Create("EPUB/" + zipName)
	err = tt.Execute(buf, e.args)
	if err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("Exec Error (%s)", filename),
		)
	}
	return nil
}

type chapterArgs struct {
	BookTitle  string
	Title      string
	Stylesheet string
	ID         string
	Content    string
}

func (e *Book) buildSection(section epubSection, sectionType string) error {
	chap := chapterArgs{
		BookTitle:  e.args.Title,
		Title:      section.title,
		Stylesheet: e.args.Stylesheet,
	}
	name := fmt.Sprintf(
		"chapter%03d-%s.xhtml",
		len(e.args.Chapters)+1,
		"%d",
	)
	tt, err := CompileTemplate("chapter.xhtml")
	if err != nil {
		return err
	}
	e.args.Chapters = append(
		e.args.Chapters,
		bookChapter{
			NavPoint: fmt.Sprintf("navPoint-%d", len(e.args.Chapters)+2),
			ID:       fmt.Sprint(len(e.args.Chapters) + 1),
			Title:    section.title,
			Path:     "text/" + fmt.Sprintf(name, 0),
			Type:     sectionType,
		},
	)

	for i, part := range section.parts {
		chap.ID = fmt.Sprintf(name, i)
		chap.Content = part

		e.args.Files = append(e.args.Files, bookFile{
			ID:        chap.ID,
			Path:      "text/" + chap.ID,
			MediaType: mtXHTML,
		})
		buf, _ := e.file.Create("EPUB/text/" + chap.ID)
		err := tt.Execute(buf, chap)
		if err != nil {
			return errors.Wrap(
				err,
				fmt.Sprintf(
					"Chapter Exec Error (%s): ",
					chap.ID,
				),
			)
		}
	}
	return nil
}
