package main

import (
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	i, err := net.Listen("tcp", ":8080") //	명령어용 소켓
	if err != nil {
		log.Println("8080서버를 열지 못함")
	}

	defer i.Close() // defer == 함수가 끝나기 전에 서버를 닫음

	for {
		conn, err := i.Accept() // i의 정보를 이용해 서버를 염
		if err != nil {
			log.Println("서버를 열지 못함")
			continue
		}
		conn.Write([]byte("서버에 접속했습니다"))

		defer conn.Close()

		go ConnHandler(conn) // connHandler함수를 비동기화 방식의 쓰레드로 실행
	}
}

func ConnHandler(conn net.Conn) {

	recvBuf := make([]byte, 124) //문자를 저장할 임시 버퍼생성
	for {
		n, err := conn.Read(recvBuf) //대기하다 클라이언트가 값을 보내면 읽어옴
		if nil != err {
			if io.EOF == err {
				log.Println("문자를 읽지 못함")
				return
			}
			log.Println("문자를 읽지 못함")
			return
		}
		if 0 < n {
			data := string(recvBuf[:n])

			if strings.Contains(data, "/ls") {
				log.Println("파일목록")
				showDirectory(conn)

			} else if strings.Contains(data, "/다운로드") {
				filepath := strings.TrimLeft(data, "/다운로드 ")
				downloadFile(conn, filepath)

			} else if strings.Contains(data, "/접속종료") {
				//endConn()

			} else if strings.Contains(data, "/로그인") {
				checkLogin(string(data), conn)
			} else if strings.Contains(data, "/업로드") {

				fileName := strings.TrimLeft(data, "/업로드") // /업로드 파일이름 + 파일사이즈 형태
				fileInfo := strings.Split(fileName, "+")   // + 기준으로 문자열을 나눠 슬라이스 형태로 저장
				fileName = fileInfo[0]                     // 파일이름 재구성
				strina := fileInfo[1]                      // 파일 사이즈 추출
				tempInt, err := strconv.Atoi(strina)       // 파일사이즈를 인트형으로
				if err != nil {
					log.Println("string to int error")
					return
				}

				fileBuf := make([]byte, tempInt) // 추출한 파일사이즈로 버퍼를 만듬
				n, _ = conn.Read(fileBuf)        // 만든버퍼에 데이터 읽기
				fileData := fileBuf[:n]

				if checkExistFile(fileName) {
					checkR := make([]byte, 30)
					conn.Write([]byte("파일이 존재합니다 덮어씌우시겠습니까? 덮어씌우려면 ^Y 아니면 ^X"))

					n, _ := conn.Read(checkR)
					checkRT := string(checkR[:n])
					checkRT = strings.TrimSuffix(checkRT, "\n")
					checkRT = strings.TrimSuffix(checkRT, "\r")

					if checkRT == "^Y" || checkRT == "^y" {
						log.Println("이미지를 다운로드 합니다")
						uploadFile(conn, fileName, fileData)
					} else {
						conn.Write([]byte("이미지를 업로드하는데 실패함"))
						continue
					}
				} else {
					uploadFile(conn, fileName, fileData)
				}
			} else {
				_, err = conn.Write([]byte(data)) // conn소켓을 가진 클라이언트에게 받은 데이터를 다시 보내줌
				if err != nil {
					log.Println("쓰기오류")
				}
			}

		}
	}
}

func checkLogin(data string, conn net.Conn) {

	idAndpw := strings.TrimLeft(data, "/로그인")
	idAndpw = strings.TrimSuffix(idAndpw, "\n")
	idAndpw = strings.TrimSuffix(idAndpw, "\r")
	onlyIdPw := strings.Split(idAndpw, "+")
	onlyIdPw[0] = strings.TrimSuffix(onlyIdPw[0], "\r")
	onlyIdPw[0] = strings.TrimSuffix(onlyIdPw[0], " ")

	log.Println(onlyIdPw[0] + "1")

	log.Println(onlyIdPw[1] + "1")

	if onlyIdPw[0] == "admin" && onlyIdPw[1] == "1234" {
		conn.Write([]byte("yes"))
		return
	}
	conn.Write([]byte("NO"))
	return
}

func showDirectory(conn net.Conn) {

	files, err := ioutil.ReadDir("./img")
	if err != nil {
		log.Println("파일목록을 불러오지 못함")
		return
	}

	var fileList string

	for _, file := range files {
		//conn.Write([]byte(file.Name()))
		n := file.Size()
		fileList = fileList + "\n" + file.Name() + "     " + strconv.Itoa(int(n))
	}
	conn.Write([]byte(fileList))
	return
}

func checkExistFile(filepath string) bool {

	filepath = strings.TrimSuffix(filepath, "\n")
	filepath = strings.TrimSuffix(filepath, "\r")
	newFilePath := "D:/studyschool/network/NetWorkFTPserver/server/img/" + filepath

	if _, err := os.Stat(newFilePath); err != nil { // 파일 존재여부
		return false
	} else {
		return true
	}
}

func uploadFile(conn net.Conn, fileName string, data []byte) {

	file, err := os.Create("./img/" + fileName) // 파일만들기
	if err != nil {
		log.Println("파일 생성 오류")
		return
	}

	n, err := file.Write(data) // 바이트형태의 데이터를 만들어놓은 파일에 쓰기
	if err != nil {
		log.Println("파일 쓰기 오류")
		return
	}

	defer file.Close()
	log.Println("정상적으로 종료됨 bytes  : ", n)
	conn.Write([]byte("정상적으로 업로드됨"))
	return
}

func downloadFile(conn net.Conn, filepath string) {

	filepath = strings.TrimSuffix(filepath, "\n")
	filepath = strings.TrimSuffix(filepath, "\r")
	newFilePath := "D:/studyschool/network/NetWorkFTPserver/server/img/" + filepath

	if fileStat, err := os.Stat(newFilePath); err != nil { // 파일 존재여부
		log.Println("error messege: ", err)
		conn.Write([]byte("파일이 존재하지 않습니다"))
		return
	} else {
		sendFile(conn, fileStat, newFilePath)
		return
	}
}

func sendFile(conn net.Conn, fileInfo fs.FileInfo, filePath string) {

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("파일을 여는데 실패함")
		return
	}

	data := make([]byte, fileInfo.Size()*1)

	n, err := file.Read(data)
	if err != nil {
		log.Println(err)
		log.Println("파일을 읽는데 실패함")
		return
	}

	defer file.Close()

	size := strconv.Itoa(n)
	sendData := "/업로드" + fileInfo.Name() + "+" + size

	log.Println(sendData)

	conn.Write([]byte("/업로드" + fileInfo.Name() + "+" + size))

	conn.Write(data)
	return
}
