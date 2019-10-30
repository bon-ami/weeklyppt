package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/bon-ami/eztools"
	"github.com/bon-ami/eztools/contacts"
)

const (
	cfgFieldUsr  = "user"
	cfgFileName  = "WeeklyPpt.cfg"
	cfgSepFields = ","
	cfgSepValues = "="
	cfgFileMode  = 0644
)

var (
	errValidityNotChecked = errors.New("Not able to check validity")
	errNotDefined         = errors.New("Not defined or configured")
	errNotSaved           = errors.New("Result not saved")
	cfgEnvUsrs            = [][]string{
		{"windows", "USERNAME"},
		{"linux", "USER"}}
)

func validateUsrID(db *sql.DB, id int) (user string, err error) {
	searched, err := eztools.Search(db, eztools.TblCONTACTS,
		eztools.FldID+"="+strconv.Itoa(id),
		[]string{eztools.FldNAME}, "")
	if err == nil {
		return searched[0][0], nil
	}
	return
}

func getCfgFileName() string {
	home, err := os.UserHomeDir()
	if err != nil {
		eztools.LogErr(err)
		return cfgFileName
	}
	return filepath.Join(home, cfgFileName)
}

//func getCfgFile(mode int) (*os.File, error) {
//return os.OpenFile(getCfgFileName(), mode, 0664)
//}

func getUsrFromSys() (name string) {
	for _, v := range cfgEnvUsrs {
		if runtime.GOOS == v[0] {
			return os.Getenv(v[1])
		}
	}
	return
}

func getUsrFromEnv(db *sql.DB) (id int, err error) {
	err = eztools.ErrNoValidResults
	for _, v := range cfgEnvUsrs {
		if runtime.GOOS == v[0] {
			id, err = contacts.GetIDFromAllNames(db,
				os.Getenv(v[1]))
			if err == nil && id == eztools.InvalidID {
				err = eztools.ErrNoValidResults
			}
			if err != nil {
				eztools.LogErr(err)
			}
			break
		}
	}
	return
}

func createHash(key string) string {
	hasher := md5.New()
	_, err := hasher.Write([]byte(key))
	if err != nil {
		eztools.LogErr(err)
		return key
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		eztools.LogErrPrint(err)
		return nil
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		eztools.LogErrPrint(err)
		return nil
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, passphrase string) []byte {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		eztools.LogErrPrint(err)
		return nil
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		eztools.LogErrPrint(err)
		return nil
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		eztools.LogErrPrint(err)
		return nil
	}
	return plaintext
}

type execCmpFieldUsr func(buf, input string) (output string, breaking bool)

func parseUsrFromFile(matched execCmpFieldUsr, unmatched execCmpFieldUsr,
	idIn int) (idOut int, output string, err error) {
	idOut = eztools.InvalidID
	buf, err := ioutil.ReadFile(getCfgFileName())
	if err != nil {
		eztools.LogErr(err)
		return
	}
	plaintext := decrypt(buf, getUsrFromSys())
	if plaintext == nil {
		err = eztools.ErrOutOfBound
		return
	}
	var (
		slc []string
	)
	/*scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//read by line
		slc = strings.Split(scanner.Text(), "=")
	slc = strings.Split(buf, "=")*/
	var execBreak bool
	idInStr := strconv.Itoa(idIn)
	for _, fields := range strings.Split(string(plaintext), cfgSepFields) {
		slc = strings.Split(fields, cfgSepValues)
		if len(slc) == 2 && slc[0] == cfgFieldUsr {
			if len(slc[1]) < 1 {
				err = errNotDefined
			} else {
				idOut, err = strconv.Atoi(slc[1])
			}
			output, execBreak = matched(output, idInStr)
		} else {
			output, execBreak = unmatched(output, fields)
		}
		if execBreak {
			break

		}
	}
	return
}

func getUsrFromfile(db *sql.DB) (id int, fixed bool, err error) {
	/*file, err := getCfgFile(os.O_RDONLY)
	if err != nil {
		return
	}
	defer file.Close()*/
	id, err = getUsrFromEnv(db)
	if err == nil {
		fixed = true
		return
	}
	id, _, err = parseUsrFromFile(func(buf,
		id string) (output string, breaking bool) {
		return "", true
	}, func(buf, id string) (output string, breaking bool) {
		return
	}, 0)
	/* if err = scanner.Err(); err != nil {
		return
	}
	if len(usr) < 1 {
		return 0, fixed, errNotDefined
	}
	id, err = strconv.Atoi(usr)*/
	return
}

//func chgUsrFromfile( /*cfg string,*/ id int) (buf string, err error) {
/*file, err := getCfgFile(os.O_CREATE | os.O_RDWR)
if err != nil {
	return
}
defer file.Close()
var (
	slc  []string
	line string
)
scanner := bufio.NewScanner(file)
for scanner.Scan() {
	line = scanner.Text()
	slc = strings.Split(line, "=")
	if len(slc) == 2 && slc[0] == cfgFieldUsr {
		slc[1] = strconv.Itoa(id)
		buf = append(buf, strings.Join(slc, "="))
	} else {
		buf = append(buf, line)
	}
}*/
/*_, buf, err = parseUsrFromFile(func(buf, idIn string) (output string, breaking bool) {
	output = buf
	if len(buf) > 0 {
		output += cfgSepFields
	}
	return output + cfgFieldUsr + cfgSepValues + idIn, false
}, func(buf, field string) (output string, breaking bool) {
	return buf + field, false
}, id)*/
/*if err = scanner.Err(); err != nil {
	return
}*/
//return
//}

func putUsrTofile(id int) (err error) {
	_, buf, err := parseUsrFromFile(func(buf,
		idIn string) (output string, breaking bool) {
		output = buf
		if len(buf) > 0 {
			output += cfgSepFields
		}
		return output + cfgFieldUsr + cfgSepValues + idIn, false
	}, func(buf, field string) (output string, breaking bool) {
		return buf + field, false
	}, id)
	//buf, err := chgUsrFromfile(id)
	if err != nil && !os.IsNotExist(err) { //err != os.ErrNotExist {
		return
	}
	if len(buf) < 1 {
		buf = cfgFieldUsr + cfgSepValues + strconv.Itoa(id)
	}

	err = ioutil.WriteFile(getCfgFileName(), encrypt([]byte(buf),
		getUsrFromSys()), cfgFileMode)
	/*file, err := getCfgFile(os.O_CREATE | os.O_TRUNC | os.O_WRONLY)
	if err != nil {
		return
	}
	defer file.Close()
	for _, line := range buf {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			break
		}
	}*/
	return
}

func chgUsr(db *sql.DB) (int, error) {
	selected, err := eztools.Search(db, eztools.TblCONTACTS, "",
		[]string{eztools.FldID, eztools.FldNAME, "ldap"}, "")
	if err != nil {
		return eztools.InvalidID, err
	}
	id := eztools.ChooseInts(selected, "Choose your ID")
	if id == eztools.InvalidID {
		return id, errNotDefined
	}
	pw := eztools.PromptPwd("Password")
	if len(pw) < 1 {
		return id, errValidityNotChecked
	}
	var ldap string
	for _, v := range selected {
		if v[0] == strconv.Itoa(id) {
			ldap = v[2]
			break
		}
	}
	err = eztools.Authenticate(db, ldap, pw)
	if err != nil {
		eztools.LogErrPrint(err)
		return id, errValidityNotChecked
	}
	err = putUsrTofile(id)
	if err != nil {
		return id, errNotSaved
	}
	return id, nil
}

func getUsr(db *sql.DB) (int, bool, error) {
	id, fixed, err := getUsrFromfile(db)
	if err == nil {
		usr, err := validateUsrID(db, id)
		if err == nil {
			eztools.ShowStr("Welcome, " + usr + ". ")
			return id, fixed, nil
		}
		return id, fixed, errValidityNotChecked
	}
	eztools.ShowStrln("No valid account set.")
	return id, fixed, err
}
