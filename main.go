package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
)

func main() {
	// Print a log message to indicate that the program is running
	fmt.Println("Logs from your program will appear here!")

	// Bind to port 4221 for listening to TCP connections
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		// Accept a connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			os.Exit(1)
		}

		// Handle the connection concurrently
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading request:", err.Error())
		return
	}
	request := string(buffer[:n])

	// Parse the request line to get the method and path
	requestLines := strings.Split(request, "\r\n")
	if len(requestLines) < 1 {
		fmt.Println("Invalid request")
		return
	}
	requestLine := requestLines[0]
	requestFields := strings.Fields(requestLine)
	if len(requestFields) < 2 {
		fmt.Println("Invalid request")
		return
	}

	// Extract headers from the request
	headers := parseHeaders(requestLines[1:])

	r := strings.Split(string(buffer), "\r\n")
	m := strings.Split(r[0], " ")[0]
	p := strings.Split(r[0], " ")[1]

	if m == "POST" && p[0:7] == "/files/" {
		content := strings.Trim(r[len(r)-1], "\x00")
		dir := os.Args[2]
		_ = os.WriteFile(path.Join(dir, p[7:]), []byte(content), 0644)
		response := "HTTP/1.1 201 Created\r\n\r\n"
		conn.Write([]byte(response))
	} else {
		if p == "/" {
			statusCode, statusText := getStatus(headers, 200, "OK")
			response := fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", statusCode, statusText)
			conn.Write([]byte(response))
		} else if strings.HasPrefix(p, "/echo/") {
			echoStr := strings.TrimPrefix(p, "/echo/")
			statusCode, statusText := getStatus(headers, 200, "OK")
			contentType := getContentType(headers, "text/plain")
			contentEncoding := getContentEncoding(headers, "")
			var response string
			if strings.Contains(contentEncoding, "gzip") {
				var b bytes.Buffer
				enc := gzip.NewWriter(&b)
				enc.Write([]byte(echoStr))
				enc.Close()
				response = fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Encoding: %s\r\nContent-Length: %d\r\n\r\n%s", statusCode, statusText, contentType, "gzip", len(b.String()), b.String())
			} else {
				response = fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s", statusCode, statusText, contentType, len(echoStr), echoStr)
			}
			conn.Write([]byte(response))
		} else if p == "/user-agent" {
			userAgent := headers["User-Agent"]
			statusCode, statusText := getStatus(headers, 200, "OK")
			contentType := getContentType(headers, "text/plain")
			response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s", statusCode, statusText, contentType, len(userAgent), userAgent)
			conn.Write([]byte(response))
		} else if strings.HasPrefix(p, "/files/") {
			dir := os.Args[2]
			fileName := strings.TrimPrefix(p, "/files/")
			fmt.Print(fileName)
			data, err := os.ReadFile(dir + fileName)
			if err != nil {
				response := "HTTP/1.1 404 Not Found\r\n\r\n"
				conn.Write([]byte(response))
			} else {
				response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(data), data)
				conn.Write([]byte(response))
			}
		} else {
			statusCode, statusText := getStatus(headers, 404, "Not Found")
			response := fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", statusCode, statusText)
			conn.Write([]byte(response))
		}
	}
}

// parseHeaders parses HTTP headers from the request lines
func parseHeaders(headerLines []string) map[string]string {
	headers := make(map[string]string)
	for _, line := range headerLines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}

// getStatus extracts the status code and status text from headers or returns default values
func getStatus(headers map[string]string, defaultCode int, defaultText string) (int, string) {
	status, ok := headers["Status"]
	if !ok {
		return defaultCode, defaultText
	}
	parts := strings.SplitN(status, " ", 2)
	if len(parts) != 2 {
		return defaultCode, defaultText
	}
	return defaultCode, parts[1]
}

// getContentType extracts the Content-Type from headers or returns a default value
func getContentType(headers map[string]string, defaultType string) string {
	contentType, ok := headers["Content-Type"]
	if !ok {
		return defaultType
	}
	return contentType
}

// getContentEncoding extracts the Accept-Encoding from headers or returns a default value
func getContentEncoding(headers map[string]string, defaultType string) string {
	contentType, ok := headers["Accept-Encoding"]
	if !ok {
		return defaultType
	}
	return contentType
}
