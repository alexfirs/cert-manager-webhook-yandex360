package yandex360api

var Yandex360ApiMock_TestData = Yandex360ApiMockSettings{
	authKey: "mockTestKey=",
	organizationsAndDomains: map[int]Domains{
		1001: {
			"example1.com": Records{
				DnsRecord{RecordID: 1, Name: "@", Type: "A", TTL: 21600, Address: "1.2.3.4"},
				DnsRecord{RecordID: 2, Name: "cname1", Type: "CNAME", TTL: 21600, Target: "someother1.site"},
				DnsRecord{RecordID: 3, Name: "sometxt1", Type: "TXT", TTL: 21600, Text: "randomtext1"},
			},
			"example2.com": Records{
				DnsRecord{RecordID: 1, Name: "@", Type: "A", TTL: 21600, Address: "4.3.2.1"},
				DnsRecord{RecordID: 2, Name: "cname2", Type: "CNAME", TTL: 21600, Target: "someother2.site"},
				DnsRecord{RecordID: 3, Name: "sometxt2", Type: "TXT", TTL: 21600, Text: "randomtext2"},
			},
		},
		1002: {
			"example3.com": Records{
				DnsRecord{RecordID: 1, Name: "@", Type: "A", TTL: 21600, Address: "8.9.10.11"},
				DnsRecord{RecordID: 2, Name: "cname3", Type: "CNAME", TTL: 21600, Target: "someother3.site"},
				DnsRecord{RecordID: 3, Name: "sometxt3", Type: "TXT", TTL: 21600, Text: "randomtext3"},
			},
		},
		1003: {
			"example.com": Records{
				DnsRecord{RecordID: 1, Name: "@", Type: "A", TTL: 21600, Address: "8.9.10.11"},
				DnsRecord{RecordID: 2, Name: "cname3", Type: "CNAME", TTL: 21600, Target: "someother3.site"},
				DnsRecord{RecordID: 3, Name: "sometxt3", Type: "TXT", TTL: 21600, Text: "randomtext3"},
			},
		},
	},
}
