package imagevalidation

import "testing"

const TEST_DBX_BIN = "dbxupdate_amd64.bin"

func TestAppendDbx(t *testing.T) {
	   // Test appending dbx to UEFI succeed without error.
	   //# Copy the bin file to the special folder.
	   //# This path only applies to ubuntu guests.
	   // runComand('sudo cp ./{TEST_DBX_BIN} /usr/share/secureboot/updates/dbx/')
}

//    self.vm.RunShellCommand('sudo chattr -i /sys/firmware/efi/efivars/dbx-*')
//    _, stdout, _ = self.vm.RunShellCommand(
//        'sudo sbkeysync --no-default-keystores --keystore '
//        '/usr/share/secureboot/updates --verbose')
//    # Output is too long. We only care about the last few lines.
//    logging.info('sbkeysync return: %s', stdout.splitlines()[-10:])
//    self.assertIn('Inserting key update', stdout)
//    self.assertNotIn('Error', stdout)
//
//    # Inssue request again to make sure the previous one succeeds.
//    _, stdout, _ = self.vm.RunShellCommand(
//        'sudo sbkeysync --no-default-keystores --keystore '
//        '/usr/share/secureboot/updates --verbose')
//    logging.info('sbkeysync return: %s', stdout.splitlines()[-10:])
//    self.assertNotIn('Inserting key update', stdout)