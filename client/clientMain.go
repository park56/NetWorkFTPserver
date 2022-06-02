package main

import (
	"bufio"
	"io/fs"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// 클라이언트가 해야할 것 :
// 1. 파일을 보내는 방법을 생각
// 2. 명령어 분리 (파일목록, 업로드 파일 경로/파일명 - 여러번 가능 - 서버에서 파일 다운로드, 접속종료)

var Conn net.Conn // conn을 전역변수로 사용하기 위해 선언

func main() {

	conn, err := net.Dial("tcp", ":8080") // 서버와 연결을 시도
	if err != nil {
		log.Println(":8080서버가 존재하지 않음")
		return
	}

	Conn = conn
	go func() { // 서버로부터 값을 읽는 반복문
		data := make([]byte, 130990)

		for {
			n, err := conn.Read(data) // 서버가 값을 던지면 읽음
			if err != nil {
				log.Println("읽기 오류")
				continue
			}

			uploadData := string(data[:n])

			if strings.Contains(uploadData, "/업로드") {
				whenDownload(uploadData)
			} else {
				log.Println("server send : " + string(data[:n])) // 읽은 데이터를 출력
				time.Sleep(time.Duration(3) * time.Second)       // 쉬기 3초간
			}
		}

	}()

	/*if login(conn) {
		log.Println("로그인 성공")
	} else {
		log.Println("로그인 실패")
		conn.Close()
	}*/

	newScanner := bufio.NewReader(os.Stdin)

	for { // 서버로 값을 넘기는 반복문
		var s string
		s, err = newScanner.ReadString('\n')
		if err != nil {
			log.Println("입력을 받는데 오류가 생김")
			continue
		}

		if strings.Contains(s, "/파일목록") { // s에 /파일목록이 포함되어 있는지 확인
			showDirectory()
		} else if strings.Contains(s, "/업로드") {
			checkFileName(s)
		} else if strings.Contains(s, "/다운로드") {
			downloadFile(conn, s)
		} else if strings.Contains(s, "/접속종료") {
			//	endConn()
		} else if strings.Contains(s, "^Y") {
			conn.Write([]byte("^Y"))
		} else if strings.Contains(s, "^X") {
			conn.Write([]byte("^X"))
		} else {
			log.Println("잘못된 명령어")
			continue
		}
		time.Sleep(time.Duration(3) * time.Second) // 3초
	}

}

func login(conn net.Conn) bool {

	newScanner := bufio.NewReader(os.Stdin)

	log.Printf("id : ")
	id, _ := newScanner.ReadString('\n')
	id = strings.TrimSuffix(id, "\n")
	log.Println()

	log.Printf("pw : ")
	pw, _ := newScanner.ReadString('\n')
	conn.Write([]byte("/로그인" + id + "+" + pw))
	result := make([]byte, 50)

	conn.Read(result)
	req := string(result)

	if strings.Contains(req, "서버에 접속했습니다") {
		log.Println("로그인 성공")
		return true
	} else {
		log.Println("로그인 실패")
		return false
	}
}

func showDirectory() {
	sentence := []byte("/ls")
	Conn.Write(sentence)
}

func checkFileName(fileInfo string) {

	/* 파일이름 재가공 */
	filepath := strings.TrimLeft(fileInfo, "/업로드 ")
	filepath = strings.TrimSuffix(filepath, "\n")
	filepath = strings.TrimSuffix(filepath, "\r")

	fileString := strings.Split(filepath, " ")
	/* 만약 파일경로에 공백이 존재한다면 */
	if len(fileString) > 2 {
		for n, v := range fileString {
			if n == len(fileString)-1 {
				break
			}
			if n == 0 {
				continue
			}
			fileString[0] += " " + v
		}
		fileString[1] = fileString[len(fileString)-1]
	}

	if fileStat, err := os.Stat(fileString[0]); err != nil { // 파일 존재여부
		log.Println("error messege: ", err)
		log.Println("파일이 존재하지 않습니다")
		return
	} else {
		if len(fileString) == 1 {
			sendFile(fileStat, fileStat.Name(), fileString[0])
			return
		} else {
			sendFile(fileStat, fileString[1], fileString[0])
			return
		}
	}
}

func sendFile(fileInfo fs.FileInfo, fileName, filePath string) {

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("파일을 여는데 실패함")
		return
	}

	data := make([]byte, fileInfo.Size())

	n, err := file.Read(data)
	if err != nil {
		log.Println(err)
		log.Println("파일을 읽는데 실패함")
		return
	}

	defer file.Close()

	size := strconv.Itoa(n)
	sendData := "/업로드" + fileName + "+" + size
	log.Println(sendData)
	Conn.Write([]byte(sendData))

	Conn.Write(data)
	return
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
	return
}

func downloadFile(conn net.Conn, filepath string) {
	conn.Write([]byte(filepath))
	return
}

func whenDownload(uploadData string) {
	fileName := strings.TrimLeft(uploadData, "/업로드") // /업로드 파일이름 + 파일사이즈 형태
	fileInfo := strings.Split(fileName, "+")         // + 기준으로 문자열을 나눠 슬라이스 형태로 저장
	fileName = fileInfo[0]                           // 파일이름 재구성
	strina := fileInfo[1]                            // 파일 사이즈 추출
	tempInt, err := strconv.Atoi(strina)             // 파일사이즈를 인트형으로
	if err != nil {
		log.Println("string to int error")
		return
	}
	fileBuf := make([]byte, tempInt) // 추출한 파일사이즈로 버퍼를 만듬
	n, _ := Conn.Read(fileBuf)       // 만든버퍼에 데이터 읽기
	fileData := fileBuf[:n]
	uploadFile(Conn, fileName, fileData)
	return
}
