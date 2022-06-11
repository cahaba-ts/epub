package epub

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var Debug = false

// FilenameAlreadyUsedError is thrown by AddCSS, AddFont, AddImage, or AddSection
// if the same filename is used more than once.
type FilenameAlreadyUsedError struct {
	Filename string // Filename that caused the error
}

func (e *FilenameAlreadyUsedError) Error() string {
	return fmt.Sprintf("Filename already used: %s", e.Filename)
}

// FileRetrievalError is thrown by AddCSS, AddFont, AddImage, or Write if there was a
// problem retrieving the source file that was provided.
type FileRetrievalError struct {
	Source string // The source of the file whose retrieval failed
	Err    error  // The underlying error that was thrown
}

func (e *FileRetrievalError) Error() string {
	return fmt.Sprintf("Error retrieving %q from source: %+v", e.Source, e.Err)
}

// Epub implements an EPUB file.
type Book struct {
	sync.Mutex
	file *zip.Writer
	buf  *bytes.Buffer

	md   goldmark.Markdown
	exts []goldmark.Extender

	// The key is the image filename, the value is the image source
	imageLookup map[string]string
	assetLookup map[string]string

	sections [3][]epubSection

	args *bookArgs
}

type bookArgs struct {
	Title          string
	Description    string
	Stylesheet     string
	StylesheetName string
	CoverImage     string
	Cover          string
	URN            string
	Author         string
	Publisher      string
	ReleaseDate    string
	CurrentDate    string
	Files          []bookFile
	Sections       []bookSection
	Chapters       []bookChapter
}
type bookFile struct {
	ID         string
	Path       string
	MediaType  string
	Properties string
}
type bookSection struct {
	Ref string
}
type bookChapter struct {
	NavPoint string
	ID       string
	Title    string
	Path     string
	Type     string
}
type epubSection struct {
	title string
	parts []string
}

// NewBook returns a new Epub.
func NewBook(title string) *Book {
	e := &Book{
		args: &bookArgs{
			Title:          title,
			Stylesheet:     "../default.css",
			StylesheetName: "default.css",
			URN:            uuid.Must(uuid.NewV4()).String(),
			CurrentDate:    time.Now().UTC().Format(time.RFC3339),
		},
		buf: &bytes.Buffer{},
		exts: []goldmark.Extender{
			extension.Table,
			extension.Strikethrough,
			extension.DefinitionList,
		},
	}
	e.file = zip.NewWriter(e.buf)

	mtf, _ := e.file.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	io.WriteString(mtf, "application/epub+zip")

	dsf, _ := e.file.CreateHeader(&zip.FileHeader{
		Name:   "OEBPS/default.css",
		Method: zip.Store,
	})
	b, _ := RetrieveTemplate("default.css")
	if _, err := dsf.Write(b); err != nil {
		log.Fatal("Write CSS: ", err)
	}
	e.args.Files = append(e.args.Files, bookFile{
		ID:        "default.css",
		Path:      "OEBPS/default.css",
		MediaType: "text/css",
	})

	cont, _ := e.file.CreateHeader(&zip.FileHeader{
		Name:   "META-INF/container.xml",
		Method: zip.Store,
	})
	b, _ = RetrieveTemplate("container.xml")
	cont.Write(b)

	e.imageLookup = make(map[string]string)
	e.assetLookup = make(map[string]string)

	return e
}

// SetCSS will set the CSS file for the book. It is not
// recommended to call this more than once for a book.
func (e *Book) SetCSS(source string) error {
	err := e.addFile("stylesheet.css", source, "text/css")
	if err != nil {
		return errors.Wrap(
			err,
			"Add CSS",
		)
	}
	e.args.Stylesheet = "../../stylesheet.css"
	e.args.StylesheetName = "stylesheet.css"
	return nil
}

func (e *Book) SetCover(source string) error {
	ext := filepath.Ext(source)
	err := e.AddImage(source, "cover"+ext)
	if err != nil {
		return err
	}
	e.args.Cover = "OEBPS/images/img_cover" + ext
	e.args.CoverImage = "../images/img_cover" + ext

	e.args.Files[len(e.args.Files)-1].Properties = "cover-image"
	return nil
}

func (e *Book) AddAsset(source, filename, mediaType string) error {
	e.Lock()
	e.assetLookup[filename] = "../assets/" + filename
	e.Unlock()
	return e.addFile("OEBPS/assets/"+filename, source, mediaType)
}

var ImageMediaTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".jxl":  "image/jxl",
	".gif":  "image/gif",
	".heif": "image/heif",
	".avif": "image/avif",
}

func (e *Book) AddImage(source, filename string) error {
	mediaType := ImageMediaTypes[filepath.Ext(source)]

	finalName := strings.ReplaceAll(filename, "/", "_")
	finalName = strings.ReplaceAll(finalName, " ", "_")
	finalName = "img_" + finalName
	e.Lock()
	if Debug {
		fmt.Println("Image Lookup: ", filename, "=>", finalName)
	}
	e.imageLookup[filename] = "../images/" + finalName
	e.Unlock()
	return e.addFile("OEBPS/images/"+finalName, source, mediaType)
}

func (e *Book) AddImageFolder(source string) error {
	return filepath.Walk(source, func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(strings.TrimPrefix(path, source), "/")
		err := e.AddImage(path, name)
		return err
	})
}

func (e *Book) addFile(zipPath, filePath, mediaType string) error {
	zipPath = strings.ReplaceAll(zipPath, " ", "_")
	e.Lock()
	defer e.Unlock()
	w, err := e.file.CreateHeader(&zip.FileHeader{
		Name:   zipPath,
		Method: zip.Store,
	})
	if err != nil {
		return err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return err
	}
	f.Close()
	e.args.Files = append(e.args.Files, bookFile{
		ID:        filepath.Base(zipPath),
		Path:      zipPath,
		MediaType: mediaType,
	})

	return nil
}

func (e *Book) LookupImage(imageFilename string) (string, bool) {
	a, b := e.imageLookup[imageFilename]
	return a, b
}
func (e *Book) LookupAsset(assetFilename string) (string, bool) {
	a, b := e.assetLookup[assetFilename]
	return a, b
}

func (e *Book) AddIntroductionMD(title string, body string) error {
	e.Lock()
	content, err := e.renderMarkdown(body)
	e.Unlock()
	if err != nil {
		return err
	}
	return e.AddIntroductionHTML(title, content)
}
func (e *Book) AddIntroductionHTML(title string, body []string) error {
	return e.addSection(0, title, body)
}
func (e *Book) AddChapterMD(title, body string) error {
	e.Lock()
	content, err := e.renderMarkdown(body)
	e.Unlock()
	if err != nil {
		return err
	}
	return e.AddChapterHTML(title, content)
}
func (e *Book) AddChapterHTML(title string, body []string) error {
	return e.addSection(1, title, body)
}
func (e *Book) AddPostscriptMD(title, body string) error {
	e.Lock()
	content, err := e.renderMarkdown(body)
	e.Unlock()
	if err != nil {
		return err
	}
	return e.AddPostscriptHTML(title, content)
}
func (e *Book) AddPostscriptHTML(title string, body []string) error {
	return e.addSection(2, title, body)
}

func (e *Book) addSection(priority int, title string, bodies []string) error {
	e.Lock()
	defer e.Unlock()
	s := epubSection{
		title: title,
		parts: bodies,
	}
	e.sections[priority] = append(e.sections[priority], s)

	return nil
}

func (e *Book) Title() string {
	return e.args.Title
}
func (e *Book) SetTitle(t string) {
	e.args.Title = t
}
func (e *Book) Author() string {
	return e.args.Author
}
func (e *Book) SetAuthor(author string) {
	e.args.Author = author
}
func (e *Book) Description() string {
	return e.args.Description
}
func (e *Book) SetDescription(d string) {
	e.args.Description = d
}
func (e *Book) Publisher() string {
	return e.args.Publisher
}
func (e *Book) SetPublisher(pub string) {
	e.args.Publisher = pub
}
func (e *Book) ReleaseDate() string {
	return e.args.ReleaseDate
}
func (e *Book) SetReleaseDate(t string) {
	e.args.ReleaseDate = t
}
func (e *Book) Identifier() string {
	return e.args.URN
}
func (e *Book) SetIdentifier(id string) {
	e.args.URN = id
}
