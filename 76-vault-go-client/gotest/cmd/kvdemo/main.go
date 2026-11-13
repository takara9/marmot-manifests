// kvdemo は、Vault を KV (v2) として使い、機密情報を保存・取得するサンプルです。
//
//	kvdemo put    marmot/db-password username=admin password=s3cret
//	kvdemo get    marmot/db-password            # 全フィールドを表示
//	kvdemo get    marmot/db-password password   # 1 フィールドだけ表示
//	kvdemo list   marmot
//	kvdemo history marmot/db-password           # バージョン一覧
//	kvdemo delete marmot/db-password            # 論理削除 (undelete で復元可)
//	kvdemo undelete marmot/db-password
//
// 接続先は環境変数 VAULT_ADDRS (または VAULT_ADDR)、VAULT_CACERT、VAULT_TOKEN で指定します。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	vaultclient "github.com/takara9/marmot-manifests/vault-go-client"
)

func main() {
	mount := flag.String("mount", "secret", "KV v2 のマウントパス")
	timeout := flag.Duration("timeout", 30*time.Second, "全体のタイムアウト")
	flag.Usage = usage
	flag.Parse()
	if err := run(*mount, *timeout, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: kvdemo [-mount secret] {put PATH KEY=VALUE... | get PATH [FIELD] | list PATH | history PATH | delete PATH | undelete PATH}")
	flag.PrintDefaults()
}

func run(mount string, timeout time.Duration, args []string) error {
	if len(args) < 2 {
		usage()
		return errors.New("command and path are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg, err := vaultclient.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.Token == "" {
		return errors.New("no token: set VAULT_TOKEN or run `vault login`")
	}
	c, err := vaultclient.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	store := vaultclient.NewStore(c, mount)
	cmd, path := args[0], args[1]

	switch cmd {
	case "put":
		data, err := parsePairs(args[2:])
		if err != nil {
			return err
		}
		sec, err := store.Put(ctx, path, data)
		if err != nil {
			return err
		}
		fmt.Printf("saved %s/%s (version %d)\n", mount, path, sec.VersionMetadata.Version)

	case "get":
		sec, err := store.Get(ctx, path)
		if err != nil {
			return err
		}
		if len(args) >= 3 {
			v, ok := sec.Data[args[2]]
			if !ok {
				return fmt.Errorf("field %q not found", args[2])
			}
			fmt.Println(v)
			return nil
		}
		fmt.Printf("# version %d, created %s\n", sec.VersionMetadata.Version, sec.VersionMetadata.CreatedTime.Format(time.RFC3339))
		for _, k := range sortedKeys(sec.Data) {
			fmt.Printf("%s=%v\n", k, sec.Data[k])
		}

	case "list":
		keys, err := store.List(ctx, path)
		if err != nil {
			return err
		}
		for _, k := range keys {
			fmt.Println(k)
		}

	case "history":
		md, err := store.Metadata(ctx, path)
		if err != nil {
			return err
		}
		fmt.Printf("current_version=%d max_versions=%d\n", md.CurrentVersion, md.MaxVersions)
		versions, err := store.Versions(ctx, path)
		if err != nil {
			return err
		}
		for _, v := range versions {
			state := "active"
			switch {
			case v.Destroyed:
				state = "destroyed"
			case !v.DeletionTime.IsZero():
				state = "deleted"
			}
			fmt.Printf("  version %d  %s  %s\n", v.Version, v.CreatedTime.Format(time.RFC3339), state)
		}

	case "delete":
		if err := store.Delete(ctx, path); err != nil {
			return err
		}
		fmt.Printf("deleted %s/%s (can be restored with undelete)\n", mount, path)

	case "undelete":
		md, err := store.Metadata(ctx, path)
		if err != nil {
			return err
		}
		if err := store.Undelete(ctx, path, md.CurrentVersion); err != nil {
			return err
		}
		fmt.Printf("restored %s/%s (version %d)\n", mount, path, md.CurrentVersion)

	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
	return nil
}

func parsePairs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, errors.New("put requires KEY=VALUE pairs")
	}
	data := make(map[string]any, len(args))
	for _, a := range args {
		k, v, ok := strings.Cut(a, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid pair %q (want KEY=VALUE)", a)
		}
		data[k] = v
	}
	return data, nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
