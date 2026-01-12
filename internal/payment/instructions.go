package payment

import "strings"

const (
	// Virtual Account
	MethodBCAVA     = "BCA_VIRTUAL_ACCOUNT"
	MethodBNIVA     = "BNI_VIRTUAL_ACCOUNT"
	MethodMandiriVA = "MANDIRI_VIRTUAL_ACCOUNT"

	// QRIS
	MethodQRIS = "QRIS"
	MethodCOD  = "COD"

	// E-Wallet
	MethodOVO     = "OVO"
	MethodDANA    = "DANA"
	MethodLINKAJA = "LINKAJA"
	MethodSHOPEE  = "SHOPEEPAY"

	// Retail Outlet
	MethodAlfamart  = "ALFAMART"
	MethodIndomaret = "INDOMARET"

	// Credit Card
	MethodCreditCard = "CREDIT_CARD"
)

var InstructionMap = map[string][]string{
	MethodCOD: {
		"Pesanan akan dikirim ke alamat tujuan",
		"Siapkan uang tunai sebesar {{amount}} saat kurir tiba",
		"Pastikan nominal pembayaran sesuai dengan total pesanan",
		"Lakukan pembayaran langsung kepada kurir",
		"Simpan bukti pembayaran dari kurir",
		"Jika tidak tersedia uang pas, siapkan nominal mendekati jumlah pembayaran",
	},

	// ========================
	// VIRTUAL ACCOUNT
	// ========================
	MethodBCAVA: {
		"Buka aplikasi BCA Mobile, KlikBCA, atau ATM BCA",
		"Pilih menu Transfer → Virtual Account",
		"Masukkan nomor Virtual Account {{payment_code}}",
		"Pastikan nama penerima dan nominal {{amount}} sudah sesuai",
		"Lakukan pembayaran dan simpan bukti transaksi",
	},

	MethodBNIVA: {
		"Buka aplikasi BNI Mobile Banking atau ATM BNI",
		"Pilih menu Virtual Account Billing",
		"Masukkan nomor Virtual Account {{payment_code}}",
		"Periksa detail pembayaran dengan nominal {{amount}}",
		"Konfirmasi dan selesaikan pembayaran",
	},

	MethodMandiriVA: {
		"Buka aplikasi Livin’ by Mandiri atau ATM Mandiri",
		"Pilih menu Bayar → Multi Payment",
		"Masukkan nomor Virtual Account {{payment_code}}",
		"Pastikan detail pembayaran dengan nominal {{amount}} sudah benar",
		"Selesaikan transaksi pembayaran",
	},

	// ========================
	// QRIS
	// ========================
	MethodQRIS: {
		"Buka aplikasi e-wallet atau mobile banking yang mendukung QRIS",
		"Pilih menu Scan / Bayar",
		"Pindai kode QR yang ditampilkan",
		"Periksa nominal pembayaran {{amount}}",
		"Konfirmasi dan selesaikan pembayaran",
	},

	// ========================
	// E-WALLET
	// ========================
	MethodOVO: {
		"Buka aplikasi OVO",
		"Pastikan saldo mencukupi untuk pembayaran {{amount}}",
		"Konfirmasi pembayaran pada notifikasi yang muncul",
		"Masukkan PIN OVO untuk menyelesaikan pembayaran",
	},

	MethodDANA: {
		"Buka aplikasi DANA",
		"Pastikan saldo DANA mencukupi untuk pembayaran {{amount}}",
		"Konfirmasi pembayaran",
		"Masukkan PIN DANA untuk menyelesaikan transaksi",
	},

	MethodLINKAJA: {
		"Buka aplikasi LinkAja",
		"Pastikan saldo mencukupi untuk pembayaran {{amount}}",
		"Konfirmasi pembayaran",
		"Masukkan PIN untuk menyelesaikan transaksi",
	},

	MethodSHOPEE: {
		"Buka aplikasi Shopee",
		"Pastikan saldo ShopeePay mencukupi untuk pembayaran {{amount}}",
		"Konfirmasi pembayaran",
		"Masukkan PIN ShopeePay",
	},

	// ========================
	// RETAIL OUTLET
	// ========================
	MethodAlfamart: {
		"Datang ke gerai Alfamart terdekat",
		"Sampaikan kepada kasir ingin melakukan pembayaran",
		"Tunjukkan kode pembayaran {{payment_code}} kepada kasir",
		"Lakukan pembayaran sesuai nominal {{amount}}",
		"Simpan struk sebagai bukti pembayaran",
	},

	MethodIndomaret: {
		"Datang ke gerai Indomaret terdekat",
		"Sampaikan kepada kasir ingin melakukan pembayaran",
		"Tunjukkan kode pembayaran {{payment_code}} kepada kasir",
		"Lakukan pembayaran sesuai nominal {{amount}}",
		"Simpan struk sebagai bukti pembayaran",
	},

	// ========================
	// CREDIT CARD
	// ========================
	MethodCreditCard: {
		"Masukkan detail kartu kredit (nomor kartu, masa berlaku, CVV)",
		"Pastikan detail kartu sudah benar",
		"Lakukan verifikasi 3D Secure (OTP dari bank penerbit)",
		"Tunggu hingga pembayaran sebesar {{amount}} berhasil diproses",
	},
}

func GetInstructions(method string) []string {
	if steps, ok := InstructionMap[method]; ok {
		return steps
	}

	return []string{
		"Ikuti instruksi pembayaran yang tersedia pada halaman ini",
	}
}

type InstructionVars map[string]string

func InjectVariables(
	steps []string,
	vars InstructionVars,
) []string {
	result := make([]string, 0, len(steps))

	for _, step := range steps {
		updated := step
		for key, value := range vars {
			updated = strings.ReplaceAll(
				updated,
				"{{"+key+"}}",
				value,
			)
		}
		result = append(result, updated)
	}

	return result
}
