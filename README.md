edit-server
===========

An edit server for TextAid and similar Google Chrome plugins.

This program provides a simple web server which accepts content as
a HTTP body in POST requests, and allows it to be edited by an external
program. The modified content is returned to the client in the HTTP
response.

It is intended to be used as an "edit server" for browser plugins such
as Google Chrome's TextAid. This is a direct port of the Perl web server
provided by the TextAid plugin's author.

This server is multi-threaded and supports multiple concurrent edits.

Install
=======

	go get github.com/sebnow/edit-server
	go install github.com/sebnow/edit-server
