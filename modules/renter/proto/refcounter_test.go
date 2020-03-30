package proto

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/NebulousLabs/fastrand"

	"gitlab.com/NebulousLabs/writeaheadlog"

	"gitlab.com/NebulousLabs/Sia/modules"

	"gitlab.com/NebulousLabs/errors"

	"gitlab.com/NebulousLabs/Sia/build"
	"gitlab.com/NebulousLabs/Sia/crypto"
	"gitlab.com/NebulousLabs/Sia/types"
)

var testWAL = newTestWAL()

// TestRefCounter_Count tests that the Count method always returns the correct
// counter value, either from disk or from in-mem storage.
func TestRefCounter_Count(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	rc := testPrepareRefCounter(2+fastrand.Uint64n(10), t)
	s := uint64(2)
	v := uint16(21)
	ov := uint16(12)

	// set up the expected value on disk
	err := writeVal(rc.filepath, s, v)
	assertSuccess(err, t, "Failed to write a count to disk:")

	// verify we can read it correctly
	rv, err := rc.Count(s)
	assertSuccess(err, t, "Failed to read count from disk:")
	if rv != v {
		t.Fatal(fmt.Sprintf("read wrong value from disk: expected %d, got %d", v, rv))
	}

	// check behaviour on bad sector number
	_, err = rc.Count(math.MaxInt64)
	assertErrorIs(err, ErrInvalidSectorNumber, t, "Expected ErrInvalidSectorNumber, got:")

	// set up a temporary override
	rc.newSectorCounts[s] = ov

	// verify we can read it correctly
	rov, err := rc.Count(s)
	assertSuccess(err, t, "Failed to read count from disk:")
	if rov != ov {
		t.Fatal(fmt.Sprintf("read wrong override value from disk: expected %d, got %d", ov, rov))
	}
}

// TestRefCounter_Append tests that the Decrement method behaves correctly
func TestRefCounter_Append(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	startNumSec := fastrand.Uint64n(10)
	rc := testPrepareRefCounter(startNumSec, t)
	stats, err := os.Stat(rc.filepath)
	assertSuccess(err, t, "RefCounter creation finished successfully but the file is not accessible:")
	err = rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")

	// test Append
	u, err := rc.Append()
	assertSuccess(err, t, "Failed to create an append update")
	if rc.numSectors != startNumSec+1 {
		t.Fatal(fmt.Errorf("Append failed to properly increase the numSectors counter. Expected %d, got %d", startNumSec+2, rc.numSectors))
	}

	// apply the update
	err = rc.CreateAndApplyTransaction(u)
	assertSuccess(err, t, "Failed to apply append update:")
	rc.UpdateApplied()

	// verify: we expect the file size to have grown by 2 bytes
	endStats, err := os.Stat(rc.filepath)
	assertSuccess(err, t, "Failed to get file stats:")
	if endStats.Size() != stats.Size()+2 {
		t.Fatal(fmt.Sprintf("File size did not grow as expected. Expected size: %d, actual size: %d", stats.Size()+2, endStats.Size()))
	}
}

// TestRefCounter_Decrement tests that the Decrement method behaves correctly
func TestRefCounter_Decrement(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	rc := testPrepareRefCounter(2+fastrand.Uint64n(10), t)
	err := rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")

	// test Decrement
	u, err := rc.Decrement(rc.numSectors - 2)
	assertSuccess(err, t, "Failed to create an decrement update:")

	// verify: we expect the value to have increased from the base 1 to 0
	readValAfterDec, err := rc.readCount(rc.numSectors - 2)
	assertSuccess(err, t, "Failed to read value after decrement:")
	if readValAfterDec != 0 {
		t.Fatal(fmt.Errorf("read wrong value after decrement. Expected %d, got %d", 2, readValAfterDec))
	}

	// check behaviour on bad sector number
	_, err = rc.Decrement(math.MaxInt64)
	assertErrorIs(err, ErrInvalidSectorNumber, t, "Expected ErrInvalidSectorNumber, got:")

	// apply the update
	err = rc.CreateAndApplyTransaction(u)
	assertSuccess(err, t, "Failed to apply decrement update:")
	rc.UpdateApplied()
}

// TestRefCounter_Delete tests that the Delete method behaves correctly
func TestRefCounter_Delete(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	rc := testPrepareRefCounter(fastrand.Uint64n(10), t)
	err := rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")

	// delete the ref counter
	u, err := rc.DeleteRefCounter()
	assertSuccess(err, t, "Failed to create a delete update")

	// apply the update
	err = rc.CreateAndApplyTransaction(u)
	assertSuccess(err, t, "Failed to apply a delete update:")
	rc.UpdateApplied()

	// verify
	_, err = os.Stat(rc.filepath)
	if err == nil {
		t.Fatal("RefCounter deletion finished successfully but the file is still on disk", err)
	}
}

// TestRefCounter_DropSectors tests that the DropSectors method behaves
// correctly and the file's size is properly adjusted
func TestRefCounter_DropSectors(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	startNumSec := 2 + fastrand.Uint64n(10)
	rc := testPrepareRefCounter(startNumSec, t)
	stats, err := os.Stat(rc.filepath)
	assertSuccess(err, t, "RefCounter creation finished successfully but the file is not accessible:")
	err = rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")

	// check behaviour on bad sector number
	// (trying to drop more sectors than we have)
	_, err = rc.DropSectors(math.MaxInt64)
	assertErrorIs(err, ErrInvalidSectorNumber, t, "Expected ErrInvalidSectorNumber, got:")

	// test DropSectors by dropping two counters
	u, err := rc.DropSectors(2)
	assertSuccess(err, t, "Failed to create truncate update:")
	if rc.numSectors != startNumSec-2 {
		t.Fatal(fmt.Errorf("wrong number of counters after Truncate. Expected %d, got %d", startNumSec-2, rc.numSectors))
	}

	// apply the update
	err = rc.CreateAndApplyTransaction(u)
	assertSuccess(err, t, "Failed to apply truncate update:")
	rc.UpdateApplied()

	//verify:  we expect the file size to have shrunk with 2*2 bytes
	endStats, err := os.Stat(rc.filepath)
	assertSuccess(err, t, "Failed to get file stats:")
	if endStats.Size() != stats.Size()-4 {
		t.Fatal(fmt.Sprintf("File size did not shrink as expected. Expected size: %d, actual size: %d", stats.Size(), endStats.Size()))
	}
}

// TestRefCounter_Increment tests that the Decrement method behaves correctly
func TestRefCounter_Increment(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	rc := testPrepareRefCounter(2+fastrand.Uint64n(10), t)
	err := rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")

	// test Increment
	u, err := rc.Increment(rc.numSectors - 2)
	assertSuccess(err, t, "Failed to create an increment update:")

	// verify: we expect the value to have increased from the base 1 to 2
	readValAfterInc, err := rc.readCount(rc.numSectors - 2)
	assertSuccess(err, t, "Failed to read value after increment:")
	if readValAfterInc != 2 {
		t.Fatal(fmt.Errorf("read wrong value after increment. Expected %d, got %d", 2, readValAfterInc))
	}

	// check behaviour on bad sector number
	_, err = rc.Increment(math.MaxInt64)
	assertErrorIs(err, ErrInvalidSectorNumber, t, "Expected ErrInvalidSectorNumber, got:")

	// apply the update
	err = rc.CreateAndApplyTransaction(u)
	assertSuccess(err, t, "Failed to apply increment update:")
	rc.UpdateApplied()
}

// TestRefCounter_Load specifically tests refcounter's Load method
func TestRefCounter_Load(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter to load
	rc := testPrepareRefCounter(fastrand.Uint64n(10), t)

	// happy case
	_, err := LoadRefCounter(rc.filepath, testWAL)
	assertSuccess(err, t, "Failed to load refcounter:")

	// fails with os.ErrNotExist for a non-existent file
	_, err = LoadRefCounter("there-is-no-such-file.rc", testWAL)
	if !errors.IsOSNotExist(err) {
		t.Fatal("Expected os.ErrNotExist, got something else:", err)
	}
}

// TestRefCounter_Load_InvalidHeader checks that loading a refcounters file with
// invalid header fails.
func TestRefCounter_Load_InvalidHeader(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare
	testContractID := types.FileContractID(crypto.HashBytes([]byte("contractId")))
	testDir := build.TempDir(t.Name())
	err := os.MkdirAll(testDir, modules.DefaultDirPerm)
	assertSuccess(err, t, "Failed to create test directory:")
	rcFilePath := filepath.Join(testDir, testContractID.String()+refCounterExtension)

	// Create a file that contains a corrupted header. This basically means
	// that the file is too short to have the entire header in there.
	f, err := os.Create(rcFilePath)
	assertSuccess(err, t, "Failed to create test file:")
	defer f.Close()

	// The version number is 8 bytes. We'll only write 4.
	_, err = f.Write(fastrand.Bytes(4))
	assertSuccess(err, t, "Failed to write to test file:")
	_ = f.Sync()

	// Make sure we fail to load from that file and that we fail with the right
	// error
	_, err = LoadRefCounter(rcFilePath, testWAL)
	assertErrorIs(err, io.EOF, t, fmt.Sprintf("Should not be able to read file with bad header, expected `%s` error, got:", io.EOF.Error()))
}

// TestRefCounter_Load_InvalidVersion checks that loading a refcounters file
// with invalid version fails.
func TestRefCounter_Load_InvalidVersion(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare
	testContractID := types.FileContractID(crypto.HashBytes([]byte("contractId")))
	testDir := build.TempDir(t.Name())
	err := os.MkdirAll(testDir, modules.DefaultDirPerm)
	assertSuccess(err, t, "Failed to create test directory:")
	rcFilePath := filepath.Join(testDir, testContractID.String()+refCounterExtension)

	// create a file with a header that encodes a bad version number
	f, err := os.Create(rcFilePath)
	assertSuccess(err, t, "Failed to create test file:")
	defer f.Close()

	// The first 8 bytes are the version number. Write down an invalid one
	// followed 4 counters (another 8 bytes).
	_, err = f.Write(fastrand.Bytes(16))
	assertSuccess(err, t, "Failed to write to test file:")
	_ = f.Sync()

	// ensure that we cannot load it and we return the correct error
	_, err = LoadRefCounter(rcFilePath, testWAL)
	assertErrorIs(err, ErrInvalidVersion, t, fmt.Sprintf("Should not be able to read file with wrong version, expected `%s` error, got:", ErrInvalidVersion.Error()))
}

// TestRefCounter_Swap tests that the Swap method results in correct values
func TestRefCounter_Swap(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	rc := testPrepareRefCounter(2+fastrand.Uint64n(10), t)
	updates := make([]writeaheadlog.Update, 0)
	err := rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")

	// increment one of the sectors, so we can tell the values apart
	uInc, err := rc.Increment(rc.numSectors - 1)
	assertSuccess(err, t, "Failed to create increment update")
	updates = append(updates, uInc)

	// test Swap
	uSwap, err := rc.Swap(rc.numSectors-2, rc.numSectors-1)
	updates = append(updates, uSwap...)
	assertSuccess(err, t, "Failed to create swap update")
	var valAfterSwap1, valAfterSwap2 uint16
	valAfterSwap1, err = rc.readCount(rc.numSectors - 2)
	assertSuccess(err, t, "Failed to read value after swap")
	valAfterSwap2, err = rc.readCount(rc.numSectors - 1)
	assertSuccess(err, t, "Failed to read value after swap")
	if valAfterSwap1 != 2 || valAfterSwap2 != 1 {
		t.Fatal(fmt.Errorf("read wrong value after swap. Expected %d and %d, got %d and %d", 2, 1, valAfterSwap1, valAfterSwap2))
	}

	// check behaviour on bad sector number
	_, err = rc.Swap(math.MaxInt64, 0)
	assertErrorIs(err, ErrInvalidSectorNumber, t, "Expected ErrInvalidSectorNumber, got:")

	// apply the updates and check the values again
	err = rc.CreateAndApplyTransaction(updates...)
	assertSuccess(err, t, "Failed to apply updates")
	rc.UpdateApplied()
}

// TestRefCounter_UpdateSessionConstraints ensures that StartUpdate() and UpdateApplied()
// enforce all applicable restrictions to update creation and execution
func TestRefCounter_UpdateSessionConstraints(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// prepare a refcounter for the tests
	rc := testPrepareRefCounter(fastrand.Uint64n(10), t)

	var u writeaheadlog.Update
	// make sure we cannot create updates outside of an update session
	_, err1 := rc.Append()
	_, err2 := rc.Decrement(1)
	_, err3 := rc.DeleteRefCounter()
	_, err4 := rc.DropSectors(1)
	_, err5 := rc.Increment(1)
	_, err6 := rc.Swap(1, 2)
	err7 := rc.CreateAndApplyTransaction(u)
	for i, err := range []error{err1, err2, err3, err4, err5, err6, err7} {
		if !errors.Contains(err, ErrUpdateWithoutUpdateSession) {
			t.Fatalf("err%v: expected %v but was %v", i+1, ErrUpdateWithoutUpdateSession, err)
		}
	}

	// start an update session
	err := rc.StartUpdate()
	assertSuccess(err, t, "Failed to start an update session")
	// delete the ref counter
	u, err = rc.DeleteRefCounter()
	assertSuccess(err, t, "Failed to create a delete update")
	// make sure we cannot create any updates after a deletion has been triggered
	_, err1 = rc.Append()
	_, err2 = rc.Decrement(1)
	_, err3 = rc.DeleteRefCounter()
	_, err4 = rc.DropSectors(1)
	_, err5 = rc.Increment(1)
	_, err6 = rc.Swap(1, 2)
	for i, err := range []error{err1, err2, err3, err4, err5, err6} {
		if !errors.Contains(err, ErrUpdateAfterDelete) {
			t.Fatalf("err%v: expected %v but was %v", i+1, ErrUpdateAfterDelete, err)
		}
	}

	// apply the update
	err = rc.CreateAndApplyTransaction(u)
	assertSuccess(err, t, "Failed to apply a delete update:")
	rc.UpdateApplied()

	// verify: make sure we cannot start an update session on a deleted counter
	err = rc.StartUpdate()
	assertErrorIs(err, ErrUpdateAfterDelete, t, "Failed to prevent an update creation after a deletion")
}

// TestRefCounter_WALFunctions tests RefCounter's functions for creating and
// reading WAL updates
func TestRefCounter_WALFunctions(t *testing.T) {
	t.Parallel()

	// test creating and reading updates
	wp := "test/wp"
	ws := uint64(2)
	wv := uint16(12)
	u := createWriteAtUpdate(wp, ws, wv)
	rp, rs, rv, err := readWriteAtUpdate(u)
	assertSuccess(err, t, "Failed to read writeAt update:")
	if wp != rp || ws != rs || wv != rv {
		t.Fatal(fmt.Sprintf("Wrong values read from WriteAt update. Expected %ws, %d, %d, found %ws, %d, %d.", wp, ws, wv, rp, rs, rv))
	}

	u = createTruncateUpdate(wp, ws)
	rp, rs, err = readTruncateUpdate(u)
	assertSuccess(err, t, "Failed to read a truncate update:")
	if wp != rp || ws != rs {
		t.Fatal(fmt.Sprintf("Wrong values read from Truncate update. Expected %ws, %d found %ws, %d.", wp, ws, rp, rs))
	}
}

// assertSuccess is a helper function that fails the test with the given message
// if there is an error
func assertSuccess(err error, t *testing.T, msg string) {
	if err != nil {
		t.Fatal(msg, err)
	}
}

// assertSuccess is a helper function that fails the test with the given message
// if there is an error
func assertErrorIs(err error, baseError error, t *testing.T, msg string) {
	if !errors.Contains(err, baseError) {
		t.Fatal(msg, err)
	}
}

// newTestWal is a helper method to create a WAL for testing.
func newTestWAL() *writeaheadlog.WAL {
	// Create the wal.
	wd := filepath.Join(os.TempDir(), "rc-wals")
	if err := os.MkdirAll(wd, modules.DefaultDirPerm); err != nil {
		panic(err)
	}
	walFilePath := filepath.Join(wd, hex.EncodeToString(fastrand.Bytes(8)))
	_, wal, err := writeaheadlog.New(walFilePath)
	if err != nil {
		panic(err)
	}
	return wal
}

// testPrepareRefCounter is a helper that creates a refcounter and fails the
// test if that is not successful
func testPrepareRefCounter(numSec uint64, t *testing.T) *RefCounter {
	tcid := types.FileContractID(crypto.HashBytes([]byte("contractId")))
	td := build.TempDir(t.Name())
	err := os.MkdirAll(td, modules.DefaultDirPerm)
	assertSuccess(err, t, "Failed to create test directory:")
	rcFilePath := filepath.Join(td, tcid.String()+refCounterExtension)
	// create a ref counter
	rc, err := NewRefCounter(rcFilePath, numSec, testWAL)
	assertSuccess(err, t, "Failed to create a reference counter:")
	return rc
}

// writeVal is a helper method that writes a certain counter value to disk. This
// method does not do any validations or checks, the caller must make certain
// that the input parameters are valid.
func writeVal(path string, secIdx uint64, val uint16) error {
	f, err := os.OpenFile(path, os.O_RDWR, modules.DefaultFilePerm)
	if err != nil {
		return errors.AddContext(err, "failed to open refcounter file")
	}
	defer f.Close()
	var b u16
	binary.LittleEndian.PutUint16(b[:], val)
	if _, err = f.WriteAt(b[:], int64(offset(secIdx))); err != nil {
		return errors.AddContext(err, "failed to write to refcounter file")
	}
	return f.Sync()
}
