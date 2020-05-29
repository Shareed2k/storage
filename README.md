# **Storage** 

storage is a Go library inspired by [Laravel File Storage](https://laravel.com/docs/7.x/filesystem) build on top of [Rclone](https://rclone.org/). Laravel provides a powerful filesystem abstraction thanks to the wonderful Flysystem PHP package by Frank de Jonge. 
So after I switched programming language to Golang, I want to have also something like file storage library for use in Golang, there is a wonderful tool called [Rclone](https://rclone.org/),
Rclone support a long list of providers, I'm gonna use Rclone as file system core lib.

## Installation

Installation is done using the `go get` command:

```bash
go get github.com/shareed2k/storage
```
