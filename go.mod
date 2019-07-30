module github.com/TruStory/octopus

require (
	cloud.google.com/go v0.43.0
	github.com/BurntSushi/toml v0.3.1
	github.com/PuerkitoBio/goquery v1.5.0 // indirect
	github.com/Sereal/Sereal v0.0.0-20190430203904-6faf9605eb56 // indirect
	github.com/TruStory/truchain v0.1.26-0.20190730151006-7ab10f51ef98
	github.com/appleboy/gofight v1.0.4 // indirect
	github.com/appleboy/gorush v1.11.1
	github.com/aws/aws-sdk-go v1.18.1
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32
	github.com/corpix/uarand v0.1.0 // indirect
	github.com/cosmos/cosmos-sdk v0.28.2-0.20190601143109-dcdabc7e6e20
	github.com/cosmos/go-bip39 v0.0.0-20180819234021-555e2067c45d // indirect
	github.com/dghubble/go-twitter v0.0.0-20190711044719-dc9b50841e5b
	github.com/dghubble/gologin v2.1.0+incompatible
	github.com/dghubble/oauth1 v0.6.0
	github.com/gernest/mention v2.0.0+incompatible
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-pg/migrations v6.7.3+incompatible
	github.com/go-pg/pg v8.0.3+incompatible
	github.com/gobuffalo/logger v1.0.1 // indirect
	github.com/gobuffalo/packr/v2 v2.5.2
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.7.2
	github.com/gorilla/securecookie v1.1.1
	github.com/graphql-go/graphql v0.7.8 // indirect
	github.com/icrowley/fake v0.0.0-20180203215853-4178557ae428
	github.com/itskingori/go-wkhtml v0.0.0-20180226001954-aa8c15cb0496
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/joho/godotenv v1.3.0
	github.com/julianshen/go-readability v0.0.0-20160929030430-accf5123e283 // indirect
	github.com/julianshen/og v0.0.0-20170124022037-897162c55567
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/matryer/is v1.2.0 // indirect
	github.com/oklog/ulid v1.3.1
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/samsarahq/go v0.0.0-20190126203740-720caea591c9 // indirect
	github.com/samsarahq/thunder v0.5.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.3.0
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tendermint/tendermint v0.31.5
	github.com/tendermint/tmlibs v0.9.0
	github.com/ugorji/go v1.1.7 // indirect
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190712062909-fae7ac547cb7 // indirect
	google.golang.org/grpc v1.22.0 // indirect
	mellium.im/sasl v0.2.1 // indirect
)

replace golang.org/x/crypto => github.com/tendermint/crypto v0.0.0-20180820045704-3764759f34a5

replace github.com/appleboy/gorush => github.com/jhernandezb/gorush v1.11.2-0.20190607192845-ca316b6313df
