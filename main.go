package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	MaxUploadSize = 10 << 30 // 10GB
)

var (
	uploadPath = ""
	ip_port    = ""
	ip         = flag.String("ip", "", "电脑路由器ip地址")
	port       = flag.Int("port", 8080, "程序运行端口,默认8080")
)

func init() {
	showIPs()

	flag.Parse()
	if *ip == "" {
		panic("ip需要指定,即电脑路由器ip地址.比如:file_kiri.exe -ip=192.168.50.223 -port=8080")
	}
	ip_port = fmt.Sprintf("%s:%d", *ip, *port)

	exe_path, err := os.Executable()
	if err != nil {
		panic(err)
	}

	current_path := filepath.Dir(exe_path)
	uploadPath = fmt.Sprintf("%s\\%s", current_path, "tmp_file")
	fmt.Printf("执行程序的路径: [%s]文件暂存路径: [%s]\n", current_path, uploadPath)

	err = os.Mkdir(uploadPath, os.ModeDir)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
}

func main() {
	http.HandleFunc("/upload", uploadFileHandler())
	fs := http.FileServer(http.Dir(uploadPath))
	http.Handle("/files/", http.StripPrefix("/files", fs))

	upload_url := fmt.Sprintf("http://%s/upload", ip_port)
	download_url := fmt.Sprintf("http://%s/files/{文件名}", ip_port)

	fmt.Printf("NOTICE: %s for upload \n%s for download\n", upload_url, download_url)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func uploadFileHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := `<html>
		<head>
			<title>局域网文件交换</title>
		</head>
		<body>
		<form enctype="multipart/form-data" action="http://%s/upload" method="post">
			<input type="file" name="uploadFile" />
			<input type="submit" value="upload" />
		</form>
		</body>
		</html>`
		html := fmt.Sprintf(s, ip_port)

		if r.Method == http.MethodGet {
			fmt.Fprint(w, html)
			return
		}

		if r.Method == http.MethodPost {
			file, fileHeader, err := r.FormFile("uploadFile")
			if err != nil {
				renderError(w, "文件读取出错", http.StatusBadRequest)
				return
			}
			defer file.Close()

			fileSize := fileHeader.Size
			fmt.Printf("文件大小: %.2fMB\n", float64(fileSize)/1024/1024)

			if fileSize > MaxUploadSize {
				renderError(w, "文件过大", http.StatusBadRequest)
				return
			}

			fileBytes, err := ioutil.ReadAll(file)
			if err != nil {
				renderError(w, "文件读取出错", http.StatusBadRequest)
				return
			}

			detectedFileType := http.DetectContentType(fileBytes)

			// fileEndings, err := mime.ExtensionsByType(detectedFileType)
			// if err != nil {
			// 	renderError(w, fmt.Sprintf("无法写入的类型%s\n", err), http.StatusInternalServerError)
			// 	return
			// }
			// fmt.Printf("文件类型:[%s]\n", fileEndings[0])

			// newPath := filepath.Join(uploadPath, fileHeader.Filename+fileEndings[0])
			newPath := filepath.Join(uploadPath, fileHeader.Filename)
			fmt.Printf("文件类型: [%s], 存储路径: [%s]\n", detectedFileType, newPath)

			newFile, err := os.Create(newPath)
			if err != nil {
				renderError(w, fmt.Sprintf("写入失败1%s\n", err), http.StatusInternalServerError)
				return
			}
			defer newFile.Close()

			if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
				renderError(w, fmt.Sprintf("写入失败2%s\n", err), http.StatusInternalServerError)
				return
			}

			runtime.GC()
			w.Write([]byte("OK"))
		}
	})
}

func showIPs() {
	fmt.Println("下面其中一个是路由器地址,选取一个作为启动参数:")
	fmt.Println("*****************************")
	defer fmt.Println("*****************************")

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println("\t", ipnet.IP.String())
			}
		}
	}
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	// statusCode
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(message))
}
