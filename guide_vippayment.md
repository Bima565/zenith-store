
avatar
 Library untuk PHP VIPayment PHP SDK.
 Kamu bingung curl ke API VIPayment? kami menerima jasa integrasi ke API VIPayment dengan biaya mulai dari Rp 100.000 (order dan status).
Game Feature
Order
Order Joki
Status
Service
Get Stock
Get Nickname Game
Webhook
Back to top
Melakukan Pemesanan
 
[POST] https://vip-reseller.co.id/api/game-feature

Parameter	Type	Value	Req.
key	string	berisi apikey anda, cek disini.	Yes
sign	string	berisi formula md5(API ID + API KEY), cek disini.	Yes
type	string	order	Yes
service	string	berisi kode layanan, cek disini.	Yes
data_no	string	berisi data id tujuan	Yes
data_zone	string	berisi data zone tujuan bila ada	No
post_additional_data	string	berisi data tambahan bila ada	No
Example Response
JSON
{
    "result": true,
    "data": {
        "trxid": "some1d",
        "data": "136216325",
        "zone": "2685",
        "service": "Mobile Legends B - 1048 Diamonds",
        "status": "waiting",
        "note": "",
        "balance": 100000,
        "price": 195695
    },
    "message": "Pesanan berhasil, pesanan akan diproses."
}
Melakukan Pemesanan Joki
 
[POST] https://vip-reseller.co.id/api/game-feature

Parameter	Type	Value	Req.
key	string	berisi apikey anda, cek disini.	Yes
sign	string	berisi formula md5 (API ID + API KEY), cek disini.	Yes
type	string	order	Yes
service	string	berisi kode layanan, cek disini.	Yes
data_no	string	berisi data email / username akun	Yes
data_zone	string	berisi data password akun	yes
additional_data	string	berisi data tambahan Login|Nickname|Hero|Catatan Pisahkan dengan |	yes
quantity	integer	jumlah pemesanan /bintang -> Joki Paket max: 1. Joki Satuan min: 3	yes
Example Response
JSON
{
    "result": true,
    "data": {
        "trxid": "some1d",
        "data": "example@gmail.com",
        "zone": "password951",
        "additional_data": "Login|Nickname|Hero|Catatan",
        "service": "Mobile Legends Joki Ranked (Satuan) - Legend \/\u2b50",
        "status": "waiting",
        "note": "",
        "balance": 100000,
        "price": 195695
    },
    "message": "Pesanan berhasil, pesanan akan diproses."
}
Melakukan Cek Status Pesanan
 
[POST] https://vip-reseller.co.id/api/game-feature

Parameter	Type	Value	Req.
key	string	berisi apikey anda.	Yes
sign	string	berisi formula md5(API ID + API KEY).	Yes
type	string	status	Yes
trxid	string	berisi id transaksi	No
limit	integer	berisi limit transaksi, hapus parameter 'trxid' jika ingin melihat banyak transaksi sekaligus.	No
Example Response
JSON
{
    "result": true,
    "data": [
        {
            "trxid": "some1d",
            "data": "136216325",
            "zone": "2685",
            "service": "Mobile Legends B - 1048 Diamonds",
            "status": "success",
            "note": "",
            "price": 195695
        }
    ],
    "message": "Detail transaksi berhasil didapatkan."
}
JSON
{
    "result": true,
    "data": [
        {
            "trxid": "some1d",
            "data": "136216325",
            "zone": "2685",
            "service": "Mobile Legends B - 1048 Diamonds",
            "status": "success",
            "note": "",
            "price": 195695
        },
        {
            "trxid": "some1d",
            "data": "136216325",
            "zone": "2600",
            "service": "Mobile Legends B - 1048 Diamonds",
            "status": "error",
            "note": "Zone tujuan salah.",
            "price": 195695
        }
    ],
    "message": "Detail transaksi berhasil didapatkan."
}
Mendapatkan Layanan
 
[POST] https://vip-reseller.co.id/api/game-feature

Parameter	Type	Value	Req.
key	string	berisi apikey anda.	Yes
sign	string	berisi formula md5(API ID + API KEY).	Yes
type	string	services	Yes
filter_game	string	berisi brand atau category yang ingin diambil. Contoh Mobile Legends atau CapCut Pro	No
filter_status	string	berisi status yang ingin di ambil available / empty	No
Example Response
JSON
{
    "result": true,
    "data": [
        {
            "code": "ML14-S14",
            "game": "Mobile Legends A",
            "name": "14 Diamonds ( 13 + 1 Bonus )",
            "price": {
                "basic": 3310,
                "premium": 3260,
                "special": 3235
            },
            "description": "14 Diamonds ( 13 + 1 Bonus )",
            "server": "1",
            "status": "empty"
        },
        {
            "code": "MLGIFT269-S1",
            "game": "Mobile Legends Gift",
            "name": "SKIN 269 Diamonds [NORMAL]",
            "price": {
                "basic": 37660,
                "premium": 36584,
                "special": 36315
            },
            "description": "
 Wajib berteman dengan akun gift setelah order cek riwayat pesanan<\/li>\r\n
 Estimasi terkirim 7-10 hari (sesuai antrian) setelah berteman<\/li>\r\n
 Dilarang ganti nickname selama belum sukses<\/li>\r\n
 Dilarang unfollow selama belum sukses<\/li>\r\n
Khusus PO SKIN hanya bisa diorder sebelum skin rilis (max H-2)<\/li>", "server": "1", "status": "available" } ], "message": "Daftar layanan berhasil didapatkan." } 
                    
Mendapatkan stock product
 
[POST] https://vip-reseller.co.id/api/game-feature

Parameter	Type	Value	Req.
key	string	berisi apikey anda.	Yes
sign	string	berisi formula md5(API ID + API KEY).	Yes
type	string	service-stock	Yes
service	string	berisi kode layanan, cek disini.	Yes
Example Response
JSON
{
    "result": true,
    "data": {
        "code": "CODE-1234",
        "name": "PRODUK 1 BULAN",
        "price": {
            "basic": 2000,
            "premium": 1500,
            "special": 1200
        },
        "description": "deskripsi produk",
        "stock": 12,
        "server": 1,
        "status": "available"
    },
    "message": "Stok tersedia."
}
Mendapatkan Nickname Game
 
[POST] https://vip-reseller.co.id/api/game-feature

Parameter	Type	Value	Req.
key	string	berisi apikey anda.	Yes
sign	string	berisi formula md5(API ID + API KEY).	Yes
type	string	get-nickname	Yes
code	string	Lihat disini	Yes
target	string	user_id	Yes
additional_target	string	zone_id	Yes
Example Response
JSON
{
    "result": true,
    "data": "Itacimo",
    "message": "Success."
}
Example Response Mobile Legends Region
JSON
{
    "result": true,
    "data": "Itacimo",
    "country": {
        "code": "ID",
        "name": "Indonesia"
    },
    "message": "Success."
}
Webhook
 
 Harap whitelist IP 178.248.73.218 .
[POST] urlcallback.com

Header	Type	Value
Content-Type	string	application/json
X-Client-Signature	string	berisi formula md5(API ID + API KEY).
Parameter	Type	Value
trxid	string	VPXXXXXXX.
data	string	xxxxxxxx.
zone	string	xxxx
service	string	service
status	string	waiting / processing / success / error
note	string	note-xxxxx
price	integer	xxxxxx
Example Request
JSON
{
    "data": {
        "trxid": "some1d",
        "data": "136216325",
        "zone": "2685",
        "service": "Mobile Legends B - 1048 Diamonds",
        "status": "success",
        "note": "",
        "price": 195695
    }
}

2019 - 2026 © PT VIPAYMENT DIGITAL SOLUTION,All rights ReservedPage Load: 0.0618s.