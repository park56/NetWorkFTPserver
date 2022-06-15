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

var Conn net.Conn     // conn을 전역변수로 사용하기 위해 선언
var CheckLogin string // 로그인을 체크하기 위한 함수 나중에 사용자 구조체만들면 거기에 포함
//const GB = 1073872000 // 1기가바이트
const GB = 130990 // golang 소켓이 안정적으로 보낼 수 있는 최대 바이트

func main() {

	CheckLogin = ""

	conn, err := net.Dial("tcp", ":8080") // 서버와 연결을 시도
	if err != nil {
		log.Println(":8080서버가 존재하지 않음")
		return
	}

	Conn = conn
	login(conn)

	go func() { // 서버로부터 값을 읽는 반복문
		data := make([]byte, 50)

		for {
			n, err := conn.Read(data) // 서버가 값을 던지면 읽음
			if err != nil {
				log.Println("읽기 오류")
				continue
			}

			uploadData := string(data[:n])

			if strings.Contains(uploadData, "/업로드") {
				whenDownload(uploadData)
			} else if strings.Contains(uploadData, "/로그인") {
				loginCheck := strings.TrimLeft(string(uploadData), "/로그인 ")
				CheckLogin = isLogin(loginCheck)
			} else {
				log.Println("server send : " + string(data[:n])) // 읽은 데이터를 출력
				time.Sleep(time.Duration(3) * time.Second)       // 쉬기 3초간
			}
		}
	}()

	/* 로그인 대기 */
	for {
		if CheckLogin != "" {
			break
		}
	}

	if CheckLogin == "로그인" {

		//go func() {

		conn.Write([]byte("/success login"))

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
				log.Println("접속을 종료합니다")
				endConn(conn)
				conn.Close()
				return
			} else if strings.Contains(s, "^Y") {
				conn.Write([]byte("^Y"))
			} else if strings.Contains(s, "^X") {
				conn.Write([]byte("^X"))
			} else {
				log.Println("잘못된 명령어")
				continue
			}
			//time.Sleep(time.Duration(3) * time.Second) // 3초
		}
		//}()
	} else {
		endConn(conn)
	}
}

// 소켓을 종료
func endConn(conn net.Conn) {
	conn.Write([]byte("/endconn"))
	return
}

// 로그인함수
func login(conn net.Conn) {

	newScanner := bufio.NewReader(os.Stdin)

	log.Println("id : ")
	id, _ := newScanner.ReadString('\n')
	id = strings.TrimSuffix(id, "\n")
	id = strings.TrimSuffix(id, "\r")

	log.Println("pw : ")
	pw, _ := newScanner.ReadString('\n')
	pw = strings.TrimSuffix(pw, "\n")
	id = strings.TrimSuffix(id, "\r")

	conn.Write([]byte("/로그인" + id + "+" + pw))
	return
}

// 로그인이 된지 확인하는 함수
func isLogin(auth string) string {
	if strings.Contains(auth, "yes") {
		log.Println("로그인 성공")
		return "로그인"
	} else {
		log.Println("로그인 실패")
		return "실패"
	}
}

// ls함수
func showDirectory() {
	sentence := []byte("/ls")
	Conn.Write(sentence)
}

// 파일이 존재하는지 확인하는 함수
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

// 파일데이터를 보내는 함수
func sendFile(fileInfo fs.FileInfo, fileName, filePath string) {

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("파일을 여는데 실패함")
		return
	}

	data := make([]byte, fileInfo.Size()*1)

	log.Println("파일을 업로드하는중")
	n, err := file.ReadAt(data, 0)
	if err != nil {
		log.Println(err)
		log.Println("파일을 읽는데 실패함")
		return
	}

	defer file.Close()

	size := strconv.Itoa(n) // int to string
	sendData := "/업로드" + fileName + "+" + size
	Conn.Write([]byte(sendData))

	if len(data) > GB {

		count := len(data) / GB

		first := 0
		second := GB

		plusValue := GB // 슬라이스의 값이 곂치지않게 더해주는값

		for i := 0; i < count; i++ {
			//log.Println("퍼스트 : ", first)
			//log.Println("세컨드 : ", second)
			Conn.Write(data[first:second])
			first += plusValue  // ex pV = 1025) 0,1025,2050...
			second += plusValue // ex) 			 1024,2049,3074...
		}
		//log.Println("마지막 보내기")
		Conn.Write(data[first:]) // 처음부터 끝까지
	} else {
		Conn.Write(data)
	}

	return
}

// 서버에서 오는 파일데이터를 업로드하는 함수
func uploadFile(conn net.Conn, fileName string, data []byte) {

	file, err := os.Create("./img/" + fileName) // 파일만들기
	if err != nil {
		log.Println("파일 생성 오류")
		return
	}

	n, err := file.WriteAt(data, 0) // 바이트형태의 데이터를 만들어놓은 파일에 쓰기
	if err != nil {
		log.Println("파일 쓰기 오류")
		return
	}

	defer file.Close()
	log.Println("정상적으로 종료됨 bytes  : ", n)
	return
}

// 다운로드를 요청하는 함수
func downloadFile(conn net.Conn, filepath string) {
	conn.Write([]byte(filepath))
	return
}

// 다운로드를 요청했을 때 나에게 오는 파일데이터를 분리
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
	fileBuf := make([]byte, tempInt*1) // 추출한 파일사이즈로 버퍼를 만듬

	temp := 0

	for {
		n, _ := Conn.Read(fileBuf[temp:]) // 만든버퍼에 데이터 읽기
		temp += n                         // 받은 데이터만큼 기준을 올림
		//log.Println("엔의 크기 : ", n)
		//log.Println("템프 의 크기 : ", temp)
		if temp >= len(fileBuf) { // 데이터를 다 받으면
			break
		}
	}
	fileData := fileBuf
	uploadFile(Conn, fileName, fileData)
	return
}
