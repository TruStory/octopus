package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("adding user_group column to the users table...")
		_, err := db.Exec(`ALTER TABLE users ADD COLUMN user_group INTEGER DEFAULT 0`)
		if err != nil {
			return err
		}
		for address, userGroup := range initialMasterList {
			_, err := db.Exec(fmt.Sprintf("UPDATE users SET user_group = %d where address='%s'", userGroup, address))
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("dropping user_group column from the users table...")
		_, err := db.Exec(`ALTER TABLE users DROP COLUMN user_group`)
		return err
	})
}

var initialMasterList = map[string]int{
	"cosmos1xqc5gwzpgdr4wjz8xscnys2jx3f9x4zy223g9w": 1,
	"cosmos1xqc5gwzpgdr4jkjkx3z9xs2x2gurgs22fzksza": 1,
	"cosmos1xqc5gwzy2ge9ysec2vursk2etqm5yjzceu04ez": 1,
	"cosmos1xqc5gwzpgdr4gk3nfdxn24jegc6rv5zewn82ch": 1,
	"cosmos1xqc5gwzpg3fyv5en2fzyx36z2se5ks33tt57e7": 1,
	"cosmos1xqc5gwzgx9z9xvjjxq6rzkfs23f9j5jx748tp4": 1,
	"cosmos1xqc5gwzpfp8ygkzdfdpnq4j3xd8y6djy5z8gfn": 1,
	"cosmos1xqc5gwzdxyuy55z8gfr4zkjhgue5xsjgcz0kgh": 1,
	"cosmos1xqc5gwz9g4pnvvznfpzyxkp5t92yx5pkx9lnsh": 1,
	"cosmos1xqc5gwfhx4f5k3jcf4r55j6ntge5y3jtxesy8r": 1,
	"cosmos1xqc5gwz3g98rgkzkx4z9q529x4grzjehfqjher": 2,
	"cosmos1xqc5gwzgxavnwdzzxef9y36hgfg9s3jc83nz8a": 2,
	"cosmos1xqc5gwzpgve9qw2g8qcyw53efcuryk22jep5n7": 2,
	"cosmos1xqc5gwzz2er4gs3389yyuwzp89z5u3jwaruq0f": 2,
	"cosmos1xqc5gwzp2ez9wjzdxet5yn22ga99w52neh8qwt": 2,
	"cosmos1xqc5gwzpg56y2kpcxarrq56gx3p9s323fdzd7u": 2,
	"cosmos1xqc5gwzzg3p5xv2p2fq4v3ekfeyy552ndl0qdl": 2,
	"cosmos1xqc5gwzggcu95dzggcun24zpfqergdf3gqmrpu": 2,
	"cosmos1xqc5gsjtf5cnzwf3gdy95njhx4f9yve4ypff25": 2,
	"cosmos1xqc5gs2e8yerz56wfqc4sdj8tpvngkjgsvh7gw": 2,
	"cosmos1xqc5gs2e8p8rz4eh2pp9jkj8fvu5gd3nuw0gh8": 2,
	"cosmos1xqc5gs3n2su4swzt8ymnwnjexpp55kj5cr7xhk": 2,
	"cosmos1xqc5gsjcgvuyys6jg9t9jvf52gmnqnfs3pmega": 2,
	"cosmos1xqc5g3zn23y9gv3hxpvyzs6kx3fysj6k0cxlc8": 2,
	"cosmos1xqc5gsesgc6nzd3cgf99gd2d29v4g5jya6c583": 2,
	"cosmos1xqc5gse3tp99w535gs64jn26ffdrz3zs7l2dlh": 2,
	"cosmos1xqc5gs692vcykv2w239yz562xdr9xw2r86l2x2": 2,
	"cosmos1xqc5gsesg5m4jv252ce9g4jgfev52s68an2ss9": 2,
	"cosmos1xqc5gsejxfxngjpkx9qn2s2txfq5uvf4lkayyk": 2,
	"cosmos1xqc5g3232p95g4j8g9drxk2rxazykvjk8632zy": 2,
	"cosmos1xqc5g32yg4yykd6xxafrvw2rx49ygv29tpy09c": 2,
	"cosmos1xqc5g3e42vun2sjdxdg9gv6hx4t5xjp3c92qle": 2,
	"cosmos1xqc5g362g4z4xd232qmyvsee2serxj6t2l3anq": 2,
	"cosmos1xqc5g36tx3ryus3323q5y5jcxguysn2xj3qnst": 2,
	"cosmos1xqc5g36gxe2yunjwx3zrx5e4g4d9zd6670vsqa": 2,
	"cosmos1xqc5g36gxdr5vsek8pg45k28tf8rx3pkjpqr08": 2,
	"cosmos1xqc5g36g2fvnvvzegegyz52jxy69gdj2ypqqzn": 2,
	"cosmos1xqc5g36ggg6ry56s299ryje4tgmyzdjng3wjvh": 2,
	"cosmos1xqc5g36gx4q45je42vmrvs3j2ym4xkz8v7dzl4": 2,
	"cosmos1xqc5g362tpq52jjt2fzysdz62ceryv62cg5676": 2,
	"cosmos1xqc5g362fd295n3kxagrvs6zg3dy2wp4d3lwg9": 2,
	"cosmos1xqc5g36tx3zyvjz2gdt9v566x9drwvpeg3hhq5": 2,
	"cosmos1xqc5g36tg3pyzjzxffpy53j9xuc9zdphwcc3sq": 2,
	"cosmos1xqc5g362t9x4ydfsger4zd3c299nj5e5qwqqcc": 2,
	"cosmos1xqc5g3622avnxd2rgdvr2d2t29f4vd6s4p5u7z": 2,
	"cosmos1xqc5g36wfpv55dp423trs33n8p2yykpkrgppr0": 2,
	"cosmos1xqc5g36sxaz9zkj5fd8rzj65xqm9sdjjmg7zw5": 2,
	"cosmos1xqc5gjzwx4grj52dtpryss6g2cm55v3ehzhetf": 2,
	"cosmos1xqc5gjz8x3yy5s3cf499x4zd8ppy53ps26xwng": 2,
	"cosmos1xqc5gjz52dv4qdjzgdp4xsfkxpz9v56d3dcw3h": 2,
	"cosmos1xqc5gjzjg3yygnj3g9z4j43cxpx5x5pjx0rd86": 2,
	"cosmos1xqc5gjzktf95sdpkx389x4zxx5c56kzhggckup": 2,
	"cosmos1xqc5gjzpxat56d3jxqmnxd6dffryzd6klxgpkj": 2,
	"cosmos1xqc5gjj5g49nx3eng399gnjx2se4y5jrh3ux3m": 2,
	"cosmos1xqc5gjzcff8rv5j9xex4j4p3xqenyjejv3868q": 2,
	"cosmos1xqc5gjz5xycygd6jxe89sw2expprsvz98cn046": 2,
	"cosmos1xqc5gjpe2pgnxs2zfvc4sd2e8y69y5jgwkwkgr": 2,
	"cosmos1xqc5gn3sxyenvjec2qm9wdjzxd2y6s3egnxr84": 2,
	"cosmos1xqc5gj6t8p8yw3zcgvuyuv2cgetnjn2x77n5df": 2,
	"cosmos1xqc5gwzr2ep5wdpjfdyyzwp4gue5sw2s09cm0x": 3,
	"cosmos1xqc5gwz9fedyg4jrgax5sdfcx5e9v3pjchamc4": 3,
	"cosmos1xqc5gwzyx9f4wdjg2fp9gvjz2ptykv35lst6aw": 3,
	"cosmos1xqc5gs6n2g645d6t89v5v46ngue9ss34g68px4": 3,
	"cosmos1xqc5g362xp9y646cf5u45v33x5m55v2wfd2cx4": 3,
}
