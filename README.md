[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/cahaba-ts/go-epub/blob/master/LICENSE)

---

__The goal of this fork is to make generation of novel epubs simple.__ 
This is not a general purpose epub library, go to bmaupin/go-epub for that. This is 
modified for the express purpose of building novel epubs.

### Features
- Forked from github.com/bmaupin/go-epub, but nearly completely rewritten
- Customizable templates for all the epub files
- Create chapters using Markdown or HTML
- Book-focused stylesheet
- Add an entire Images Folder instead of individual files

For an example of actual usage, see https://github.com/cahaba-ts/cahaba

### Contributions

I'm not interested in generalizing the library in a way that complicates 
the novel workflow. Tweaks to the css file or fixes for bugs in specific 
epub applications would be helpful.

### Development

Clone this repository using Git, it requires Go version 1.18 or higher.

Dependencies are managed using [Go modules](https://golang.org/ref/mod)
