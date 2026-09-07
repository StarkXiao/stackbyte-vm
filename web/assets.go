package web
import "embed"
// Files contains the complete browser console and requires no runtime files.
//
//go:embed index.html styles.css app.js
var Files embed.FS
