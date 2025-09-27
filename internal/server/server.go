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
)

const TestFile1 =  "D:/Elden Ring Nightreign [DODI Repack]/data1.doi"
const TestFile2 = "C:/Users/prana_zhfhs6u/Downloads/parsec-windows.exe"

func StartTcpServer(killSwitch chan os.Signal) {
	listener,err := net.Listen("tcp","127.0.0.1:8000")
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
			log.Printf("Accepted connection from %s", conn.RemoteAddr())
			go readLoop(conn, &wg)
		}
	}()
	<-killSwitch
	log.Println("Shutdown signal received, closing listener...")
	listener.Close()
	wg.Wait()
	log.Println("All connections closed. Server gracefully shut down.")
}

func readLoop(conn net.Conn, wg *sync.WaitGroup){
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
		progressWriter := utils.NewProgressWriter(fileSize, "Receiving")
		// Create a MultiWriter to write to both the file and the progress bar.
		destWriter := io.MultiWriter(outFile, progressWriter)

		n, err := io.CopyN(destWriter,conn,fileSize)
		if err != nil{
			log.Printf("Error during file copy: %v", err)
		}
		fmt.Printf("Successfully saved file %s (%d bytes).\n", outFile.Name(), n)

	}
	
}

func SendFile(filePath string) error{
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



	conn , err := net.Dial("tcp",":8000")
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

	progressWriter := utils.NewProgressWriter(fileSize, "Transferring")

	reader := io.TeeReader(bufferedReader, progressWriter)

	n, err := io.Copy(conn, reader)
	if err!= nil{
		log.Fatalln("couldnt copy to conn from buffer.",err)
	}
	log.Println("bytes sent to conn: ",n)
	return nil
}