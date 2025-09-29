package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"shareIt/internal/utils"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

const TestFile1 =  "D:/Elden Ring Nightreign [DODI Repack]/data1.doi"
const TestFile2 = "C:/Users/prana_zhfhs6u/Downloads/parsec-windows.exe"

func StartTcpServer(killSwitch chan os.Signal, port int, p *tea.Program) {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	var wg sync.WaitGroup

	go func(){
		for{
			conn , err := listener.Accept()
			if err != nil {
				log.Println("listener err:", err)
				return
			}
			
			wg.Add(1)
			p.Send(utils.LogMsg{Message: fmt.Sprintf("Accepted connection from %s", conn.RemoteAddr())})
			go readLoop(conn, &wg, p)
		}
	}()
	<-killSwitch
	log.Println("Shutdown signal received, closing listener...")
	listener.Close()
	wg.Wait()
	log.Println("All connections closed. Server gracefully shut down.")
}

func readLoop(conn net.Conn, wg *sync.WaitGroup, p *tea.Program){
	defer conn.Close()
	defer wg.Done()
	for{

		// Read the filename length
		var filenameLength int64
		err := binary.Read(conn, binary.LittleEndian, &filenameLength)
		if err != nil {
			log.Printf("Error reading filename length: %v", err)
			return
		}

		// Read the filename
		filenameBytes := make([]byte, filenameLength)
		_, err = io.ReadFull(conn, filenameBytes)
		if err != nil {
			log.Printf("Error reading filename: %v", err)
			return
		}
		filename := string(filenameBytes)
		log.Printf("Received filename header: %s", filename)

		//Read the file content size
		var fileSize int64
		err = binary.Read(conn,binary.LittleEndian,&fileSize)
		if err != nil {
			log.Printf("Error reading file size: %v", err)
			return
		}
		log.Printf("Received file size header: %d bytes", fileSize)

		outFile, err := os.Create(filename)
		if err != nil {
			log.Printf("Error creating destination file: %v", err)
			return
		}
		defer outFile.Close()

		// Create a progress writer to track the download.
		progressWriter := utils.NewProgressWriter(fileSize, filename, "Receiving", p)
		// Create a MultiWriter to write to both the file and the progress bar.
		destWriter := io.MultiWriter(outFile, progressWriter)

		_, err = io.CopyN(destWriter,conn,fileSize)
		if err != nil{
			log.Printf("Error during file copy: %v", err)
		}
		p.Send(utils.LogMsg{Message: fmt.Sprintf("Successfully saved file %s", outFile.Name())})

	}
	
}

func SendFile(filePath string,peerAddress string, p *tea.Program){
	f , err :=os.Open(filePath)
	if err!= nil{
		log.Fatal("err opeing a file.",err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		log.Fatal("could not get file info", err)
	}
	fileSize := fileInfo.Size()
	filename := filepath.Base(filePath)
	filenameLength := int64(len(filename))



	conn , err := net.Dial("tcp",peerAddress)
	if err!=nil{
		log.Fatal(err)
	}

	//Send the filename length
	err = binary.Write(conn, binary.LittleEndian, filenameLength)
	if err != nil {
		log.Fatalf("could not write filename length to conn: %v", err)
	}
	// send filename
	_, err = conn.Write([]byte(filename))
	if err != nil {
		log.Fatalf("could not write filename to conn: %v", err)
	}
	//send file size
	err = 	binary.Write(conn, binary.LittleEndian, fileSize)
	if err != nil {
		log.Fatalf("could not write file size to conn: %v", err)
	}

	bufferedReader := bufio.NewReader(f)

	progressWriter := utils.NewProgressWriter(fileSize, filename, "Sending", p)

	reader := io.TeeReader(bufferedReader, progressWriter)

	_, err = io.Copy(conn, reader)
	if err!= nil{
		log.Fatalln("couldnt copy to conn from buffer.",err)
	}
	p.Send(utils.LogMsg{Message: fmt.Sprintf("Finished sending %s", filename)})
}