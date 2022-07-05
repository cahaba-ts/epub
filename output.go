package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

var (
	mtXHTML = "application/xhtml+xml"
	mtNCX   = "application/x-dtbncx+xml"
	mtOPF   = "application/opf+xml"
)

func (e *Book) Write(filename string) error {
	fmt.Println("Building: ", filename)
	// write cover.xhtml
	err := e.execTemplate("cover.xhtml", "OEBPS/text/cover.xhtml", mtXHTML)
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
	if err := e.execTemplate("nav.xhtml", "OEBPS/text/nav.xhtml", mtXHTML); err != nil {
		return err
	}
	e.args.Files[len(e.args.Files)-1].Properties = "nav"
	//if err := e.execTemplate("toc.ncx", "OEBPS/toc.ncx", mtNCX); err != nil {
	//	return err
	//}

	// write book.opf
	if err := e.execTemplate("content.opf", "OEBPS/content.opf", mtOPF); err != nil {
		return err
	}
	e.file.Flush()

	if err := e.file.Close(); err != nil {
		return errors.Wrap(
			err,
			"Close zip file",
		)
	}
	f, err := os.Create("temp.epub")
	if err != nil {
		return err
	}
	_, err = io.Copy(f, e.buf)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	fmt.Println("Finalizing: ", filename)
	cmd := exec.Command("ebook-polish", "-i", "-u", "temp.epub", filename)
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "epub-polish error")
	}
	return os.Remove("temp.epub")
}

func (e *Book) execTemplate(filename, zipName, mediaType string) error {
	tt, err := CompileTemplate(filename)
	if err != nil {
		return err
	}
	id := filepath.Base(zipName)
	if id == "toc.ncx" {
		id = "ncx"
	}
	if id == "nav.xhtml" {
		id = "nav"
	}
	e.args.Files = append(e.args.Files, bookFile{
		ID:        id,
		Path:      zipName,
		MediaType: mediaType,
	})
	buf, _ := e.file.Create(zipName)
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
	Header     bool
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
			NavPoint: fmt.Sprintf("navPoint%d", len(e.args.Chapters)+2),
			ID:       fmt.Sprint(len(e.args.Chapters) + 1),
			Title:    section.title,
			Path:     "OEBPS/text/" + fmt.Sprintf(name, 0),
			Type:     sectionType,
		},
	)

	for i, part := range section.parts {
		chap.ID = fmt.Sprintf(name, i)
		chap.Content = part
		chap.Header = i == 0

		e.args.Files = append(e.args.Files, bookFile{
			ID:        chap.ID,
			Path:      "OEBPS/text/" + chap.ID,
			MediaType: mtXHTML,
		})
		e.args.Sections = append(e.args.Sections, bookSection{Ref: chap.ID})
		buf, _ := e.file.CreateHeader(&zip.FileHeader{
			Name:   "OEBPS/text/" + chap.ID,
			Method: zip.Store,
		})
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
