/* The MIT License (MIT)
 *
 * Copyright (c) 2014 Sebastian Nowicki
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
 * BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
 * ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

/* This program provides a simple web server which accepts content as
 * a HTTP body in a POST request and allows it to be edited by an
 * external source. The modified content is returned to the client in
 * the HTTP response.
 *
 * It is intended to be used as an "edit server" for browser plugins
 * such as Google Chrome's TextAid. This is a direct port of the Perl
 * web server provided by the TextAid plugin's author.
 *
 * This server is multi-threaded and supports multiple concurrent edits.
 */
package main

import (
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "os/exec"
    "strings"

    log "github.com/cihub/seelog"
)

type EditHandler struct {
    // Whether an Origin header is required.
    RequireExtensionOrigin bool
    // The command to be executed for editing, with args.
    EditorCmd string
}

func (h *EditHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    if req.Method != "POST" {
        fmt.Fprintf(w, "Server is up and running.  To use it, issue a POST request with the file to edit as the content body.\n")
        return
    }

    err := h.authorise(req)
    if err != nil {
        log.Warnf("Unauthorised: %s", err)
        w.WriteHeader(401)
        fmt.Fprintf(w, "Unauthorized: %s\n", err)
        return
    }

    file, err := ioutil.TempFile("", "edit-server-XXXX")
    if err != nil {
        log.Errorf("Unable to open temporary file: %s", err)
        w.WriteHeader(500)
        return
    }
    log.Debugf("Created temporary file '%s'", file.Name())
    defer func(fname string) {
        log.Debugf("Unlinking temporary file '%s'", fname)
        os.Remove(fname)
    }(file.Name())

    log.Debugf("Writing content to '%s': '%s'", file.Name(), req.Body)
    bytesWritten, err := writeContentToFile(req, file)
    file.Sync()
    file.Close()

    if int64(bytesWritten) != req.ContentLength {
        log.Errorf("Unable to write full content (%d bytes), only %d bytes written", req.ContentLength, bytesWritten)
        w.WriteHeader(500)
        return
    }
    log.Debugf("Wrote %d bytes to '%s'", bytesWritten, file.Name())

    editorArgs := strings.Split(h.EditorCmd, " ")
    editorArgs = append(editorArgs, file.Name())
    log.Debugf("%s", editorArgs)

    log.Infof("Running editor %s %s", h.EditorCmd, file.Name())
    editor := exec.Command(editorArgs[0], editorArgs[1:]...)
    editor.Run()

    returnedContent, err := ioutil.ReadFile(file.Name())
    if err != nil {
        log.Errorf("Unable to read returned content: %s", err)
        w.WriteHeader(500)
        return
    }

    log.Debug("Returning content")
    w.Write(returnedContent)
    return
}

func (h *EditHandler) authorise(req *http.Request) error {
    if h.RequireExtensionOrigin {
        origin := req.Header.Get(("Origin"))
        if !strings.HasPrefix(origin, "chrome-extension:") {
            return fmt.Errorf("%s", "unauthorized origin")
        }
    }
    return nil
}

func writeContentToFile(req *http.Request, file *os.File) (n int, err error) {
    content := make([]byte, req.ContentLength)
    bytesRead, err := req.Body.Read(content)
    if err != nil || int64(bytesRead) != req.ContentLength {
        return
    }

    n, err = file.Write(content)
    return
}

func main() {
    var host = flag.String("b", ":8888", "Bind address")
    var editorCommand = flag.String("c", "gvim -f", "The editor command")
    flag.Parse()

    editHandler := &EditHandler{
        RequireExtensionOrigin: true,
        EditorCmd:              *editorCommand,
    }

    log.Infof("Binding edit server on %s", *host)
    http.Handle("/", http.Handler(editHandler))
    err := http.ListenAndServe(*host, nil)
    if err != nil {
        log.Criticalf("ListenAndServe: %s", err)
    }
}
