// Copyright (c) 2014, Robert Clausecker <fuzxxl@gmail.com>
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU Lesser General Public License as published by the
// Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>

package freefare

/*
#include <stdlib.h>
#include <string.h>

// workaround: type is a reserved keyword, but mifare_desfire_version_info
// contains a member named type. Let's rename it to avoid trouble
#define type type_
#include <freefare.h>
#undef type

// auxilliary typedefs to ease the implementation of
// DESFireTag.DESFireFileSettings
typedef struct {
	uint32_t file_size;
} standard_file;

typedef struct {
	int32_t lower_limit;
	int32_t upper_limit;
	int32_t limited_credit_value;
	uint8_t limited_credit_enabled;
} value_file;

typedef struct {
	uint32_t record_size;
	uint32_t max_number_of_records;
	uint32_t current_number_of_records;
} linear_record_file;
*/
import "C"
import "strconv"
import "unsafe"

// DESFire cryptography modes. Compute the bitwise or of these constants and the
// key number to select a certain cryptography mode.
const (
	CRYPTO_DES    = 0x00
	CRYPTO_3K3DES = 0x40
	CRYPTO_AES    = 0x80
)

// Convert a Tag into an DESFireTag to access functionality available for
// Mifare DESFire tags.
type DESFireTag struct {
	*tag
}

// Get last PCD error. This function wraps mifare_desfire_last_pcd_error(). If
// no error has occured, this function returns nil.
func (t DESFireTag) LastPCDError() error {
	err := Error(C.mifare_desfire_last_pcd_error(t.ctag))
	if err == 0 {
		return nil
	} else {
		return err
	}
}

// Get last PICC error. This function wraps mifare_desfire_last_picc_error(). If
// no error has occured, this function returns nil.
func (t DESFireTag) LastPICCError() error {
	err := Error(C.mifare_desfire_last_picc_error(t.ctag))
	if err == 0 {
		return nil
	} else {
		return err
	}
}

// Figure out what kind of error is hidden behind an EIO. This function largely
// replicates the behavior of freefare_strerror().
func (t DESFireTag) resolveEIO() error {
	err := t.Device().LastError()
	if err != nil {
		return err
	}

	err = t.LastPCDError()
	if err != nil {
		return err
	}

	err = t.LastPICCError()
	if err != nil {
		return err
	}

	return Error(UNKNOWN_ERROR)
}

// Connect to a Mifare DESFire tag. This causes the tag to be active.
func (t DESFireTag) Connect() error {
	r, err := C.mifare_desfire_connect(t.ctag)
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Disconnect from a Mifare DESFire tag. This causes the tag to be inactive.
func (t DESFireTag) Disconnect() error {
	r, err := C.mifare_desfire_disconnect(t.ctag)
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Authenticate to a Mifare DESFire tag. Notice that this wrapper does not
// provide wrappers for the mifare_desfire_authenticate_iso() and
// mifare_desfire_authenticate_aes() functions as the key type can be deducted
// from the key.
func (t DESFireTag) Authenticate(keyNo byte, key DESFireKey) error {
	r, err := C.mifare_desfire_authenticate(t.ctag, C.uint8_t(keyNo), key.key)
	if r == 0 {
		return nil
	}

	return t.TranslateError(err)
}

// Change the selected application settings to s. The application number of keys
// cannot be changed after the application has been created.
func (t DESFireTag) ChangeKeySettings(s byte) error {
	r, err := C.mifare_desfire_change_key_settings(t.ctag, C.uint8_t(s))
	if r == 0 {
		return nil
	}

	return t.TranslateError(err)
}

// Return the key settings and maximum number of keys for the selected
// application.
func (t DESFireTag) KeySettings() (settings, maxKeys byte, err error) {
	var s, mk C.uint8_t
	r, err := C.mifare_desfire_get_key_settings(t.ctag, &s, &mk)
	if r != 0 {
		return 0, 0, t.TranslateError(err)
	}

	settings = byte(s)
	maxKeys = byte(mk)
	err = nil
	return
}

// Change the key keyNo from oldKey to newKey. Depending on the application
// settings, a previous authentication with the same key or another key may be
// required.
func (t DESFireTag) ChangeKey(keyNo byte, newKey, oldKey DESFireKey) error {
	r, err := C.mifare_desfire_change_key(t.ctag, C.uint8_t(keyNo), newKey.key, oldKey.key)
	if r == 0 {
		return nil
	}

	return t.TranslateError(err)
}

// Retrieve the version of the key keyNo for the selected application.
func (t DESFireTag) KeyVersion(keyNo byte) (byte, error) {
	var version C.uint8_t
	r, err := C.mifare_desfire_get_key_version(t.ctag, C.uint8_t(keyNo), &version)
	if r != 0 {
		return 0, t.TranslateError(err)
	}

	return byte(version), nil
}

// Create a new application with AID aid, settings and keyNo authentication
// keys. Authentication keys are set to 0 after creation. This wrapper does not
// wrap the functions mifare_desfire_create_application_3k3des() and
// mifare_desfire_create_application_aes(). Or keyNo with the constants
// CRYPTO_3K3DES and CRYPTO_AES instead.
func (t DESFireTag) CreateApplication(aid DESFireAid, settings, keyNo byte) error {
	r, err := C.mifare_desfire_create_application(
		t.ctag, aid.cptr(), C.uint8_t(settings), C.uint8_t(keyNo))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Create a new application with AID aid, settings, keyNo authentication keys,
// and, if wantIsoFileIdentifiers is true, an ISO file ID and an optional file
// name isoFileName. This wrapper does not wrap the functions
// mifare_desfire_create_application_3k3des_iso and
// mifare_desfire_create_application_aes_iso(). Or keyNo with the constants
// CRYPTO_3K3DES and CRYPTO_AES instead.
func (t DESFireTag) CreateApplicationIso(
	aid DESFireAid,
	settings byte,
	keyNo byte,
	wantIsoFileIdentifiers bool,
	isoFileId uint16,
	isoFileName []byte,
) error {
	wifi := C.int(0)
	if wantIsoFileIdentifiers {
		wifi = 1
	}

	r, err := C.mifare_desfire_create_application_iso(
		t.ctag,
		aid.cptr(),
		C.uint8_t(settings),
		C.uint8_t(keyNo),
		wifi,
		C.uint16_t(isoFileId),
		(*C.uint8_t)(&isoFileName[0]),
		C.size_t(len(isoFileName)))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Delete the application identified by aid
func (t DESFireTag) DeleteApplication(aid DESFireAid) error {
	r, err := C.mifare_desfire_delete_application(t.ctag, aid.cptr())
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

//mifare_desfire_get_application_ids (MifareTag tag, MifareDESFireAID *aids[], size_t *count);

// Return a list of all applications of the card
func (t DESFireTag) ApplicationIds() ([]DESFireAid, error) {
	var count C.size_t
	var caids *C.MifareDESFireAID
	r, err := C.mifare_desfire_get_application_ids(t.ctag, &caids, &count)
	if r != 0 {
		return nil, t.TranslateError(err)
	}

	aids := make([]DESFireAid, int(count))
	aidsptr := uintptr(unsafe.Pointer(caids))
	for i := range aids {
		// Assume that a C.MifareDESFireAID is a *[3]C.uint8_t
		aidptr := (*DESFireAid)(unsafe.Pointer(aidsptr + uintptr(i)*unsafe.Sizeof(*caids)))
		aids[i] = *aidptr
	}

	C.mifare_desfire_free_application_ids(caids)
	return aids, nil
}

// A Mifare DESFire directory file
type DESFireDF struct {
	DESFireAid
	Fid  uint16 // file ID
	Name []byte // no longer than 16 bytes
}

// Retrieve a list of directory file (df) names
func (t DESFireTag) DFNames() ([]DESFireDF, error) {
	var count C.size_t
	var cdfs *C.MifareDESFireDF
	r, err := C.mifare_desfire_get_df_names(t.ctag, &cdfs, &count)
	if r != 0 {
		return nil, t.TranslateError(err)
	}

	dfs := make([]DESFireDF, int(count))
	dfsptr := uintptr(unsafe.Pointer(cdfs))
	for i := range dfs {
		dfptr := (*C.MifareDESFireDF)(unsafe.Pointer(dfsptr + uintptr(i)*unsafe.Sizeof(*cdfs)))
		dfs[i] = DESFireDF{
			NewDESFireAid(uint32(dfptr.aid)),
			uint16(dfptr.fid),
			C.GoBytes(unsafe.Pointer(&dfptr.df_name[0]), C.int(dfptr.df_name_len)),
		}
	}

	C.free(unsafe.Pointer(dfsptr))
	return dfs, nil
}

// Select an application. After Connect(), the master application is selected.
// This function can be used to select a different application.
func (t DESFireTag) SelectApplication(aid DESFireAid) error {
	r, err := C.mifare_desfire_select_application(t.ctag, aid.cptr())
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Reset t to factory defaults. For this function to work, a previous
// authentication with the card master key is required. WARNING: This function
// is irreversible and will delete all date on the card.
func (t DESFireTag) FormatPICC() error {
	r, err := C.mifare_desfire_format_picc(t.ctag)
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Version information for a Mifare DESFire tag.
type DESFireVersionInfo struct {
	Hardware, Software struct {
		VendorId                   byte
		Type, Subtype              byte
		VersionMajor, VersionMinor byte
		StorageSize                byte
		Protocol                   byte
	}

	Uid                            [7]byte
	BatchNumber                    [5]byte
	ProductionWeek, ProductionYear byte
}

// Retrieve various information about t including UID. batch number, production
// date, hardware and software information.
func (t DESFireTag) Version() (DESFireVersionInfo, error) {
	var ci C.struct_mifare_desfire_version_info
	r, err := C.mifare_desfire_get_version(t.ctag, &ci)
	if r != 0 {
		return DESFireVersionInfo{}, t.TranslateError(err)
	}

	vi := DESFireVersionInfo{}

	vih := &vi.Hardware
	vih.VendorId = byte(ci.hardware.vendor_id)
	vih.Type = byte(ci.hardware.type_)
	vih.Subtype = byte(ci.hardware.subtype)
	vih.VersionMajor = byte(ci.hardware.version_major)
	vih.VersionMinor = byte(ci.hardware.version_minor)
	vih.StorageSize = byte(ci.hardware.storage_size)
	vih.Protocol = byte(ci.hardware.protocol)

	vis := &vi.Software
	vis.VendorId = byte(ci.software.vendor_id)
	vis.Type = byte(ci.software.type_)
	vis.Subtype = byte(ci.software.subtype)
	vis.VersionMajor = byte(ci.software.version_major)
	vis.VersionMinor = byte(ci.software.version_minor)
	vis.StorageSize = byte(ci.software.storage_size)
	vis.Protocol = byte(ci.software.protocol)

	for i := range vi.Uid {
		vi.Uid[i] = byte(ci.uid[i])
	}

	for i := range vi.BatchNumber {
		vi.BatchNumber[i] = byte(ci.batch_number[i])
	}

	vi.ProductionWeek = byte(ci.production_week)
	vi.ProductionYear = byte(ci.production_year)

	return vi, nil
}

// Get the amount of free memory on the PICC of a Mifare DESFire tag in bytes.
func (t DESFireTag) FreeMem() (uint32, error) {
	var size C.uint32_t
	r, err := C.mifare_desfire_free_mem(t.ctag, &size)
	if r != 0 {
		return 0, t.TranslateError(err)
	}

	return uint32(size), nil
}

// This function can be used to deactivate the format function or to switch
// to use a random UID.
func (t DESFireTag) SetConfiguration(disableFormat, enableRandomUid bool) error {
	// Notice that bool is a macro. the actual type is named _Bool.
	r, err := C.mifare_desfire_set_configuration(
		t.ctag, C._Bool(disableFormat), C._Bool(enableRandomUid))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Replace the ATS bytes returned by the PICC when it is selected. This function
// performs the following extra test in order to ensure memory safety:
//
//     if len(ats) < int(ats[0]) {
//         return Error(PARAMETER_ERROR)
//     }
func (t DESFireTag) SetAts(ats []byte) error {
	// mifare_desfire_set_ats reads ats[0] bytes out of ats, so it better
	// had be that long.
	if len(ats) < int(ats[0]) {
		return Error(PARAMETER_ERROR)
	}

	r, err := C.mifare_desfire_set_ats(t.ctag, (*C.uint8_t)(&ats[0]))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Get the card's UID. This function can be used to get the original UID of the
// target if the PICC is configured to return a random UID. The return value of
// CardUID() has the same format as the return value of UID(), but this function
// may fail.
func (t DESFireTag) CardUID() (string, error) {
	var cstring *C.char
	r, err := C.mifare_desfire_get_card_uid(t.ctag, &cstring)
	defer C.free(unsafe.Pointer(cstring))
	if r != 0 {
		return "", t.TranslateError(err)
	}

	return C.GoString(cstring), nil
}

// Return a list of files in the selected application
func (t DESFireTag) FileIds() ([]byte, error) {
	var cfiles *C.uint8_t
	var count C.size_t
	r, err := C.mifare_desfire_get_file_ids(t.ctag, &cfiles, &count)
	defer C.free(unsafe.Pointer(cfiles))
	if r != 0 {
		return nil, t.TranslateError(err)
	}

	return C.GoBytes(unsafe.Pointer(cfiles), C.int(count)), nil
}

// Return a list of ISO file identifiers
func (t DESFireTag) IsoFileIds() ([]uint16, error) {
	var cfiles *C.uint16_t
	var count C.size_t
	r, err := C.mifare_desfire_get_iso_file_ids(t.ctag, &cfiles, &count)
	defer C.free(unsafe.Pointer(cfiles))
	if r != 0 {
		return nil, t.TranslateError(err)
	}

	// Cutting corners here
	ids := make([]uint16, int(count))
	C.memcpy(unsafe.Pointer(&ids[0]), unsafe.Pointer(cfiles), count)

	return ids, nil
}

// DESFire file types as used in DESFireFileSettings
const (
	STANDARD_DATA_FILE = iota
	BACKUP_DATA_FILE
	VALUE_FILE_WITH_BACKUP
	LINEAR_RECORD_FILE_WITH_BACKUP
	CYCLIC_RECORD_FILE_WITH_BACKUP
)

// Mifare DESFire communication modes
const (
	PLAIN      = 0x00
	MACED      = 0x01
	ENCIPHERED = 0x03
)

// Mifare DESFire access rights. This wrapper does not provide the constants
// MDAR_KEY0 ... MDAR_KEY13 as they are just 0 ... 13.
const (
	FREE = 0xe
	DENY = 0xf
)

// Create an uint16 out of individual access rights. This function only looks
// at the low nibbles of each parameter. This function implements the
// functionality of the MDAR macro from the freefare.h header.
func MakeDESFireAccessRights(read, write, readWrite, changeAccessRights byte) uint16 {
	ar := uint16(read&0xf) << 12
	ar |= uint16(write&0xf) << 8
	ar |= uint16(readWrite&0xf) << 4
	ar |= uint16(changeAccessRights&0xf) << 0
	return ar
}

// Split an access rights block into individual access rights. This function
// implements the functionality of the MDAR_### family of macros.
func SplitDESFireAccessRights(ar uint16) (read, write, readWrite, changeAccessRights byte) {
	read = byte(ar >> 12 & 0xf)
	write = byte(ar >> 8 & 0xf)
	readWrite = byte(ar >> 4 & 0xf)
	changeAccessRights = byte(ar >> 0 & 0xf)
	return
}

// This type remodels struct mifare_desfire_file_settings. Because Go does not
// support union types, this struct contains all union members laid out
// sequentially. Only the set of members denoted by FileType is valid. Use the
// supplied constants for FileType.
//
// Use the function SplitDESFireAccessRights() to split the AccessRights field.
type DESFireFileSettings struct {
	FileType              byte
	CommunicationSettings byte
	AccessRights          uint16

	// FileType == STANDARD_DATA_FILE || FileType == BACKUP_DATA_FILE
	FileSize uint32

	// FileType == VALUE_FILE_WITH_BACKUP
	LowerLimit, UpperLimit int32
	LimitedCreditValue     int32
	LimitedCreditEnabled   byte

	// FileType == LINEAR_RECORD_FILE_WITH_BACKUP || CYCLIC_RECORD_FILE_WITH_BACKUP
	RecordSize             uint32
	MaxNumberOfRecords     uint32
	CurrentNumberOfRecords uint32
}

// Retrieve the settings of the file fileNo of the selected application of t.
func (t DESFireTag) DESFireFileSettings(fileNo byte) (DESFireFileSettings, error) {
	var cfs C.struct_mifare_desfire_file_settings
	r, err := C.mifare_desfire_get_file_settings(t.ctag, C.uint8_t(fileNo), &cfs)
	if r != 0 {
		// explicitly invalid FileType. Behavior is subject to change.
		return DESFireFileSettings{FileType: 0xff}, t.TranslateError(err)
	}

	fs := DESFireFileSettings{
		FileType:              byte(cfs.file_type),
		CommunicationSettings: byte(cfs.communication_settings),
		AccessRights:          uint16(cfs.access_rights),
	}

	sptr := unsafe.Pointer(&cfs.settings[0])
	switch fs.FileType {
	case STANDARD_DATA_FILE:
		fallthrough
	case BACKUP_DATA_FILE:
		sf := (*C.standard_file)(sptr)
		fs.FileSize = uint32(sf.file_size)

	case VALUE_FILE_WITH_BACKUP:
		vf := (*C.value_file)(sptr)
		fs.LowerLimit = int32(vf.lower_limit)
		fs.UpperLimit = int32(vf.upper_limit)
		fs.LimitedCreditValue = int32(vf.limited_credit_value)
		fs.LimitedCreditEnabled = byte(vf.limited_credit_enabled)

	case LINEAR_RECORD_FILE_WITH_BACKUP:
		fallthrough
	case CYCLIC_RECORD_FILE_WITH_BACKUP:
		lrf := (*C.linear_record_file)(sptr)
		fs.RecordSize = uint32(lrf.record_size)
		fs.MaxNumberOfRecords = uint32(lrf.max_number_of_records)
		fs.CurrentNumberOfRecords = uint32(lrf.current_number_of_records)

	default:
		panic("Unexpected file type " + strconv.Itoa(int(fs.FileType)))
	}

	return fs, nil
}

// Change the communication settings and access rights of file fileNo of the
// selected application of t. Use the function MakeDESFireAccessRights() to
// create a suitable accessRights parameter.
func (t DESFireTag) ChangeFileSettings(fileNo, communicationSettings byte, accessRights uint16) error {
	r, err := C.mifare_desfire_change_file_settings(
		t.ctag, C.uint8_t(fileNo), C.uint8_t(communicationSettings), C.uint16_t(accessRights))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Create a standard or backup data file of size fileSize. This function wraps
// either mifare_desfire_create_std_data_file() or
// mifare_desfire_create_backup_data_file() depending on the value of isBackup.
func (t DESFireTag) CreateDataFile(
	fileNo byte,
	communicationSettings byte,
	accessRights uint16,
	fileSize uint32,
	isBackup bool,
) error {
	var r C.int
	var err error

	if isBackup {
		r, err = C.mifare_desfire_create_std_data_file(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(fileSize))
	} else {
		r, err = C.mifare_desfire_create_backup_data_file(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(fileSize))
	}

	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Create a standard or backup data file of size fileSize with an ISO file ID.
// This function wraps either mifare_desfire_create_std_data_file_iso() or
// mifare_desfire_create_backup_data_file_iso() depending on the value of
// isBackup.
func (t DESFireTag) CreateDataFileIso(
	fileNo byte,
	communicationSettings byte,
	accessRights uint16,
	fileSize uint32,
	isoFileId uint16,
	isBackup bool,
) error {
	var r C.int
	var err error

	if isBackup {
		r, err = C.mifare_desfire_create_std_data_file_iso(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(fileSize),
			C.uint16_t(isoFileId))
	} else {
		r, err = C.mifare_desfire_create_backup_data_file_iso(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(fileSize),
			C.uint16_t(isoFileId))
	}

	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Create a value file of value value constrained in the range lowerLimit to
// upperLimit and with the limitedCreditEnable settings.
func (t DESFireTag) CreateValueFile(
	fileNo byte,
	communicationSettings byte,
	accessRights uint16,
	lowerLimit, upperLimit, value int32,
	limitedCreditEnable byte,
) error {
	r, err := C.mifare_desfire_create_value_file(
		t.ctag, C.uint8_t(fileNo),
		C.uint8_t(communicationSettings),
		C.uint16_t(accessRights), C.int32_t(lowerLimit),
		C.int32_t(upperLimit), C.int32_t(value),
		C.uint8_t(limitedCreditEnable))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Create linear or cyclic record file that can holf maxNumberOfRecords of size
// recordSize. This function wraps either
// mifare_desfire_create_linear_record_file() or
// mifare_desfire_create_cyclic_record_file() depending on the value of the
// isCyclic parameter.
func (t DESFireTag) CreateRecordFile(
	fileNo byte,
	communicationSettings byte,
	accessRights uint16,
	recordSize uint32,
	maxNumberOfRecords uint32,
	isCyclic bool,
) error {
	var r C.int
	var err error
	if isCyclic {
		r, err = C.mifare_desfire_create_cyclic_record_file(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(recordSize),
			C.uint32_t(maxNumberOfRecords))
	} else {
		r, err = C.mifare_desfire_create_linear_record_file(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(recordSize),
			C.uint32_t(maxNumberOfRecords))
	}

	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Create linear or cyclic record file that can holf maxNumberOfRecords of size
// recordSize with an ISO file ID. This function wraps either
// mifare_desfire_create_linear_record_file_iso() or
// mifare_desfire_create_cyclic_record_file_iso() depending on the value of the
// isCyclic parameter.
func (t DESFireTag) CreateRecordFileIso(
	fileNo byte,
	communicationSettings byte,
	accessRights uint16,
	recordSize uint32,
	maxNumberOfRecords uint32,
	isoFileId uint16,
	isCyclic bool,
) error {
	var r C.int
	var err error
	if isCyclic {
		r, err = C.mifare_desfire_create_cyclic_record_file_iso(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(recordSize),
			C.uint32_t(maxNumberOfRecords), C.uint16_t(isoFileId))
	} else {
		r, err = C.mifare_desfire_create_linear_record_file_iso(
			t.ctag, C.uint8_t(fileNo),
			C.uint8_t(communicationSettings),
			C.uint16_t(accessRights), C.uint32_t(recordSize),
			C.uint32_t(maxNumberOfRecords), C.uint16_t(isoFileId))
	}

	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}

// Remove the file fileNo from the selected application
func (t DESFireTag) DeleteFile(fileNo byte) error {
	r, err := C.mifare_desfire_delete_file(t.ctag, C.uint8_t(fileNo))
	if r != 0 {
		return t.TranslateError(err)
	}

	return nil
}
