package main

import (
	"C"
	"archive/zip"
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/disintegration/imaging" // Import the imaging package
	"github.com/dslipak/pdf"
	"github.com/poorny/docconv"

	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)
import (
	"runtime"
	"sync"
)

func main() {}

//export ResizeImage

//export ResizeImage
func ResizeImage(hexString *C.char, x *C.char, y *C.char) *C.char {
	imageData, err := hex.DecodeString(C.GoString(hexString))
	if err != nil {
		return C.CString(fmt.Sprintf("Failed to decode hex string: %v", err))
	}

	decodedImg, format, err := image.Decode(strings.NewReader(string(imageData)))
	if err != nil {
		return C.CString(fmt.Sprintf("Failed to decode image: %v", err))
	}

	aspectRatio := float64(decodedImg.Bounds().Dx()) / float64(decodedImg.Bounds().Dy())

	xInt, err := strconv.Atoi(C.GoString(x))
	if err != nil {
		return C.CString(fmt.Sprintf("Failed to convert 'x' to int: %v", err))
	}

	yInt, err := strconv.Atoi(C.GoString(y))
	if err != nil {
		return C.CString(fmt.Sprintf("Failed to convert 'y' to int: %v", err))
	}

	var newWidth, newHeight int
	if aspectRatio > 1 {
		newWidth = xInt
		newHeight = int(math.Round(float64(xInt) / aspectRatio))
	} else {
		newWidth = int(math.Round(float64(yInt) * aspectRatio))
		newHeight = yInt
	}

	if newWidth < xInt {
		newWidth = xInt
		newHeight = int(math.Round(float64(xInt) / aspectRatio))
	}
	if newHeight < yInt {
		newHeight = yInt
		newWidth = int(math.Round(float64(yInt) * aspectRatio))
	}

	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup
	wg.Add(numCPU)

	stripHeight := decodedImg.Bounds().Dy() / numCPU
	resizedStrips := make([]image.Image, numCPU)

	for i := 0; i < numCPU; i++ {
		go func(i int) {
			defer wg.Done()

			startY := i * stripHeight
			endY := startY + stripHeight
			if i == numCPU-1 {
				endY = decodedImg.Bounds().Dy()
			}

			strip := imaging.Crop(decodedImg, image.Rect(0, startY, decodedImg.Bounds().Dx(), endY))
			resizedStripHeight := int(math.Round(float64(endY-startY) * float64(newHeight) / float64(decodedImg.Bounds().Dy())))
			resizedStrip := imaging.Resize(strip, newWidth, resizedStripHeight, imaging.Lanczos)
			resizedStrips[i] = resizedStrip
		}(i)
	}

	wg.Wait()

	finalImage := imaging.New(newWidth, newHeight, image.Transparent)
	currentY := 0
	for _, strip := range resizedStrips {
		finalImage = imaging.Paste(finalImage, strip, image.Pt(0, currentY))
		currentY += strip.Bounds().Dy()
	}

	var resizedBuffer bytes.Buffer
	var encodeErr error

	if format == "jpeg" {
		encodeErr = jpeg.Encode(&resizedBuffer, finalImage, nil)
	} else if format == "png" {
		encodeErr = png.Encode(&resizedBuffer, finalImage)
	} else {
		return C.CString(fmt.Sprintf("Unsupported image format: %s", format))
	}

	if encodeErr != nil {
		return C.CString(fmt.Sprintf("Failed to encode resized image: %v", encodeErr))
	}

	resizedHexString := hex.EncodeToString(resizedBuffer.Bytes())

	return C.CString(resizedHexString)
}

// ======================================================================
//
//export ExtractImageFromZipAsHex
func ExtractImageFromZipAsHex(zipFile *C.char, imagePath *C.char) *C.char {
	zipPath := C.GoString(zipFile)
	imagePathStr := C.GoString(imagePath)

	r, _ := zip.OpenReader(zipPath)

	defer r.Close()

	for _, f := range r.File {
		if f.Name == imagePathStr {
			rc, err := f.Open()
			if err != nil {
				return C.CString("") //fmt.Sprintf("Failed to open image file: %s", err))
			}
			defer rc.Close()

			imageData, err := ioutil.ReadAll(rc)
			if err != nil {
				return C.CString("") //fmt.Sprintf("Failed to read image file: %s", err))
			}

			hexString := hex.EncodeToString(imageData)
			return C.CString(hexString)
		}
	}

	return C.CString("") //fmt.Sprintf("Image file '%s' not found in '%s'", imagePathStr, zipPath))
}

// ======================================================================
//
//export PDF2txt
func PDF2txt(name *C.char) *C.char {

	inputFile := C.GoString(name)

	outputtext := convertPDFToText(inputFile)
	return C.CString(outputtext)
}

func convertPDFToText(inputFile string) string {
	f, err := os.Open(inputFile)
	if err != nil {
		return ("failed to open input file")
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return ("failed to get file information")
	}
	fileSize := stat.Size()

	pdfReader, err := pdf.NewReader(f, fileSize)
	if err != nil {
		return ("failed to open PDF ")
	}

	numPages := pdfReader.NumPage()
	finaltext := ""
	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := pdfReader.Page(pageNum)

		text, err := page.GetPlainText(nil)
		finaltext = finaltext + text
		if err != nil {
			return ("failed to extract text from page")
		}

		fmt.Println("====================")
		fmt.Printf("Page %d:\n", pageNum)
		fmt.Println("====================")
		fmt.Println(text)
	}

	return finaltext
}

// ======================================================================
//
//export Docx2txt
func Docx2txt(name *C.char) *C.char {
	// Open the Word document
	file, err := os.Open(C.GoString(name))
	if err != nil {
		C.CString("Failed to open document")
	}
	defer file.Close()

	// Extract the text from the Word document
	text, _, err := docconv.ConvertDoc(file)
	if err != nil {
		C.CString("Failed to read document")
	}
	return C.CString(text)

}

// ======================================================================
//
//export Speedtest
//export Speedtest
//export Speedtest
func Speedtest() *C.char {
	// URL of the file to download for speed test
	fileURL := "https://ash-speed.hetzner.com/100MB.bin"
	startTime := time.Now()

	// Perform the download
	resp, err := http.Get(fileURL)
	if err != nil {
		return C.CString("Failed to download file")
	}
	defer resp.Body.Close()

	// Read the response body to calculate the download speed
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return C.CString("Failed to read response body")
	}

	// Calculate the elapsed time in seconds
	elapsed := time.Since(startTime).Seconds()

	// Calculate the download speed in Mbps
	fileSize := resp.ContentLength
	downloadSpeed := (float64(fileSize) * 8) / (elapsed * 1000000)

	// Print the download speed
	return C.CString(fmt.Sprintf("Download speed: %.2f Mbps\n", downloadSpeed))
}

// ======================================================================
//
//export CurrentUsername
func CurrentUsername() *C.char {
	var username [256]uint16
	size := uint32(len(username))

	// Retrieve the username of the currently logged-on user
	err := syscall.GetUserNameEx(syscall.NameSamCompatible, &username[0], &size)
	if err != nil {
		errorMessage := "Failed to get the current username"
		return C.CString(errorMessage)
	}

	// Convert the UTF-16 encoded username to a Go string
	goUsername := syscall.UTF16ToString(username[:])
	username2 := goUsername
	re := regexp.MustCompile(`\\(.+)$`)
	matches := re.FindStringSubmatch(username2)
	if len(matches) > 1 {
		username2 = matches[1]
	}

	return C.CString(username2)
}

// ======================================================================
//
//export GetDefaultAdapterMacAddress
func GetDefaultAdapterMacAddress() *C.char {
	interfaces, err := net.Interfaces()
	if err != nil {
		return C.CString("")
	}

	for _, iface := range interfaces {
		// Check if the interface is up and not a loopback interface
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			// Retrieve the hardware (MAC) address
			macAddr := iface.HardwareAddr.String()
			if macAddr != "" {
				return C.CString(macAddr) //, nil
			}
		}
	}

	return C.CString("Failed to find default network adapter MAC address") //, fmt.Errorf("failed to find default network adapter MAC address")
}

// ======================================================================
//
//export GetHostname
func GetHostname() *C.char {
	hostname, err := os.Hostname()
	if err != nil {
		return C.CString("")
	}

	return C.CString(hostname)
}

// ======================================================================
//
//export ExternalIP
func ExternalIP() *C.char {
	resp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		C.CString("Failed to retrieve IP address:")
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		C.CString("Failed to read response:")
	}
	return C.CString(string(ip))
}

// ======================================================================
//
//export LocalIP
func LocalIP() *C.char {
	ifaces, err := net.Interfaces()
	if err != nil {
		return C.CString("Failed to retrieve network interfaces:")

	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			C.CString("Failed to retrieve addresses for interface")
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}

			if ipnet.IP.To4() != nil {
				return C.CString(ipnet.IP.String())
			}
		}
	}
	return C.CString("Failed")
}

// ======================================================================
//
//export ResolveIP
func ResolveIP(name *C.char) *C.char {
	names, err := net.LookupAddr(C.GoString(name))
	if err != nil {
		return C.CString("Error")
	}

	// Return the first domain name found
	if len(names) > 0 {
		return C.CString(names[0])
	}

	return C.CString("Error")
}

// ======================================================================
//
//export Ping
func Ping(name *C.char) *C.char {
	ipAddress := C.GoString(name) // Replace with the remote address you want to ping

	elapsed, err := Ping2(ipAddress)
	if err != nil {
		//fmt.Printf("Ping to %s failed: %s\n", ipAddress, err)
		return C.CString("Error")
	}

	//fmt.Printf("Ping to %s succeeded. Time taken: %s\n", ipAddress, elapsed)
	return C.CString(elapsed.String())

}

// ICMPHeader represents the ICMP header structure
type ICMPHeader struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	// Identifier and SequenceNumber fields are used for identifying the ping request and response
	Identifier     uint16
	SequenceNumber uint16
}

// Ping sends an ICMP echo request to the specified IP address and returns the elapsed time
func Ping2(ipAddress string) (time.Duration, error) {
	conn, err := net.Dial("ip4:icmp", ipAddress)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	msg := ICMPHeader{
		Type:           8, // Echo request
		Code:           0,
		Checksum:       0,
		Identifier:     uint16(os.Getpid() & 0xffff),
		SequenceNumber: 1,
	}

	msgBytes, err := encodeICMPHeader(msg)
	if err != nil {
		return 0, err
	}

	// Calculate checksum
	msg.Checksum = checksum(msgBytes)

	// Encode ICMP header again with the correct checksum
	msgBytes, err = encodeICMPHeader(msg)
	if err != nil {
		return 0, err
	}

	start := time.Now()

	_, err = conn.Write(msgBytes)
	if err != nil {
		return 0, err
	}

	reply := make([]byte, 1024)
	err = conn.SetReadDeadline(time.Now().Add(3 * time.Second)) // Timeout after 3 seconds
	if err != nil {
		return 0, err
	}

	_, err = conn.Read(reply)
	if err != nil {
		return 0, err
	}

	elapsed := time.Since(start)
	return elapsed, nil
}

// Encode the ICMP header structure to bytes
func encodeICMPHeader(header ICMPHeader) ([]byte, error) {
	buffer := make([]byte, 8)

	buffer[0] = header.Type
	buffer[1] = header.Code
	buffer[2] = 0
	buffer[3] = 0

	// Big-endian order for Checksum
	buffer[4] = byte(header.Checksum >> 8)
	buffer[5] = byte(header.Checksum)

	buffer[6] = byte(header.Identifier >> 8)
	buffer[7] = byte(header.Identifier)

	return buffer, nil
}

// Calculate the ICMP header checksum
func checksum(buffer []byte) uint16 {
	var sum uint32
	length := len(buffer)
	index := 0

	for length > 1 {
		sum += uint32(buffer[index])<<8 | uint32(buffer[index+1])
		index += 2
		length -= 2
	}

	if length > 0 {
		sum += uint32(buffer[index]) << 8
	}

	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}

	return uint16(^sum)
}

// *
//
//export HelloWorld
func HelloWorld(name *C.char) *C.char {
	who := C.GoString(name)
	return C.CString("Hello " + who)
}

//export HelloWorld2
func HelloWorld2(name *C.char) *C.char {
	who := C.GoString(name)
	return C.CString("Hello " + who)
}
