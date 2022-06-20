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
	"time"
)

var socketList map[string]net.Conn

const GB = 130990

func main() {

	socketList = map[string]net.Conn{}

	i, err := net.Listen("tcp", ":5060") //	명령어용 소켓
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

		socketList[conn.LocalAddr().String()] = conn // 맵(딕셔너리)에 소켓주소-소켓의 형태로 저장

		defer conn.Close()

		log.Println(conn.LocalAddr().String() + " 주소의 소켓 생성")

		go ConnHandler(socketList[conn.LocalAddr().String()]) // connHandler함수를 비동기화 방식의 쓰레드로 실행
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
				log.Println(data)
				filepath := strings.TrimLeft(data, "/다운로드 ")
				downloadFile(conn, filepath)

			} else if strings.Contains(data, "/endconn") {
				conn.Write([]byte("소켓을 종료합니다"))
				log.Println(conn.LocalAddr().String() + " 주소의 소켓을 종료합니다")
				conn.Close()
				delete(socketList, conn.LocalAddr().String())
			} else if strings.Contains(data, "/업로드") {

				log.Println(data)
				fileName := strings.TrimLeft(data, "/업로드") // /업로드 파일이름 + 파일사이즈 형태
				fileInfo := strings.Split(fileName, "+")   // + 기준으로 문자열을 나눠 슬라이스 형태로 저장
				fileName = fileInfo[0]                     // 파일이름 재구성
				strina := fileInfo[1]                      // 파일 사이즈 추출

				tempInt, err := strconv.Atoi(strina) // 파일사이즈를 인트형으로
				if err != nil {
					log.Println("string to int error")
					return
				}

				fileBuf := make([]byte, tempInt*1) // 추출한 파일사이즈로 버퍼를 만듬

				temp := 0 // 데이터를 받을 때 기준이 될 변수

				for {
					n, _ = conn.Read(fileBuf[temp:]) // 만든버퍼에 데이터 읽기
					/*if n <= 0 || (err != nil) {        // 클라이언트가 강제로 종료되었을 때
						log.Println("다운로드중 문제가 발생")
						return
					}*/
					temp += n // 받은 데이터만큼 기준을 올림
					//log.Println("엔의 크기 : ", n)
					//log.Println("템프 의 크기 : ", temp)
					if temp >= len(fileBuf) { // 데이터를 다 받으면
						break
					}
				}

				fileData := fileBuf

				if checkExistFile(fileName) { // 이미 파일이 존재하는 경우
					checkR := make([]byte, 30)
					conn.Write([]byte("파일이 존재합다 덮어씌우시겠습니까? 덮어씌우려면 ^Y 아니면 ^X"))

					n, _ := conn.Read(checkR)
					checkRT := string(checkR[:n])
					checkRT = strings.TrimSuffix(checkRT, "\n")
					checkRT = strings.TrimSuffix(checkRT, "\r")

					if checkRT == "^Y" || checkRT == "^y" {
						log.Println("파일을 다운로드 합니다")
						uploadFile(conn, fileName, fileData)
					} else {
						conn.Write([]byte("이미지를 업로드하는데 실패함"))
						continue
					}
				} else {
					uploadFile(conn, fileName, fileData)
				}
			} else if strings.Contains(data, "/로그인") {
				checkLogin(string(data), conn)
			} else if strings.Contains(data, "/success login") {
				log.Println(conn.LocalAddr().String() + " 로그인 성공")
			} else {
				_, err = conn.Write([]byte(data)) // conn소켓을 가진 클라이언트에게 받은 데이터를 다시 보내줌
				if err != nil {
					log.Println("쓰기오류")
				}
			}
		}
	}
}

// 아이디와 패스워드를 확인하는 함수
func checkLogin(data string, conn net.Conn) {

	idAndpw := strings.TrimLeft(data, "/로그인")
	idAndpw = strings.TrimSuffix(idAndpw, "\n")
	idAndpw = strings.TrimSuffix(idAndpw, "\r")
	onlyIdPw := strings.Split(idAndpw, "+")
	onlyIdPw[0] = strings.TrimSuffix(onlyIdPw[0], "\r")
	onlyIdPw[0] = strings.TrimSuffix(onlyIdPw[0], " ")

	if onlyIdPw[0] == "admin" && onlyIdPw[1] == "1234" {
		conn.Write([]byte("/로그인 yes"))
		return
	}
	conn.Write([]byte("/로그인 NO"))
	return
}

// ls를 담당하는 함수
func showDirectory(conn net.Conn) {

	files, err := ioutil.ReadDir("./img")
	if err != nil {
		log.Println("파일목록을 불러오지 못함")
		return
	}

	var fileList string

	if len(files) == 0 {
		conn.Write([]byte("파일이 존재하지 않습니다"))
	}

	for _, file := range files {
		//conn.Write([]byte(file.Name()))
		n := file.Size()
		fileList = file.Name() + "     " + strconv.Itoa(int(n))
		conn.Write([]byte(fileList))
		time.Sleep(time.Second * 1)
	}
	return
}

// 파일이 존재하는지 확인하는 함수
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

// 클라에서 보내는 파일데이터를 업로드하는 함수
func uploadFile(conn net.Conn, fileName string, data []byte) {

	file, err := os.Create("./img/" + fileName) // 파일만들기
	if err != nil {
		log.Println("파일 생성 오류")
		return
	}

	n, err := file.WriteAt(data, 0) // 바이트형태의 데이터를 만들어놓은 파일에 쓰기
	if err != nil {
		log.Println("파일 쓰기 오류")
		deleteFile("./img/" + fileName)
		return
	}

	defer file.Close()
	defer func() {
		if !fileSizeCheck("./img/"+fileName, n) {
			deleteFile("./img/" + fileName)
		}
	}()

	log.Println("정상적으로 종료됨 bytes  : ", n)
	conn.Write([]byte("정상적으로 업로드됨"))
	return
}

// 클라이언트에서 다운로드를 요청할 때 파일이 존재하는지 확인
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

// 클라이언트한테 파일을 보내는 함수
func sendFile(conn net.Conn, fileInfo fs.FileInfo, filePath string) {

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("파일을 여는데 실패함")
		return
	}

	data := make([]byte, fileInfo.Size()*1)

	n, err := file.ReadAt(data, 0)
	if err != nil {
		log.Println(err)
		log.Println("파일을 읽는데 실패함")
		return
	}

	defer file.Close()

	size := strconv.Itoa(n)

	conn.Write([]byte("/업로드" + fileInfo.Name() + "+" + size))

	if len(data) > GB {

		count := len(data) / GB

		first := 0
		second := GB

		plusValue := GB // 슬라이스의 값이 곂치지않게 더해주는값

		for i := 0; i < count; i++ {

			conn.Write(data[first:second])
			first += plusValue
			second += plusValue
		}
		conn.Write(data[first:]) // 처음부터 끝까지
	} else {
		conn.Write(data)
	}
	return
}

// 파일사이즈를 체크하는 함수
func fileSizeCheck(fileName string, fileSize int) bool {

	fileStat, err := os.Stat(fileName)
	if err != nil {
		log.Println("파일사이즈체크 : 존재하지 않는 파일")
		return false
	}

	if fileStat.Size() != int64(fileSize) {
		log.Println("파일업로드 사이즈와 올라간 사이즈가 다름")
		return false
	} else {
		log.Println("정상적으로 업로드 됨")
		return true
	}
}

// 파일을 삭제하는 함수
func deleteFile(fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		log.Println("딜리트파일 : 파일을 삭제하는데 실패함")
		return
	}
	log.Println("딜리트파일 : 파일을 삭제하는데 성공함")
	return
}
