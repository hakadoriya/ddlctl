# [ddlctl](https://github.com/kunitsucom/ddlctl)

`ddlctl` is a tool to control RDBMS DDLs: output all RDBMS DDLs, generate DDLs from tagged Golang source code, view differences between RDBMS and your DDL, and automate migrations.

> [!WARNING]
> This project is experimental. It is operational in the author's environment, but it is not known if it can be operated in other environments without trouble.  

[![license](https://img.shields.io/github/license/kunitsucom/ddlctl)](LICENSE)
[![pkg](https://pkg.go.dev/badge/github.com/kunitsucom/ddlctl)](https://pkg.go.dev/github.com/kunitsucom/ddlctl)
[![goreportcard](https://goreportcard.com/badge/github.com/kunitsucom/ddlctl)](https://goreportcard.com/report/github.com/kunitsucom/ddlctl)
[![workflow](https://github.com/kunitsucom/ddlctl/workflows/go-lint/badge.svg)](https://github.com/kunitsucom/ddlctl/tree/main)
[![workflow](https://github.com/kunitsucom/ddlctl/workflows/go-test/badge.svg)](https://github.com/kunitsucom/ddlctl/tree/main)
[![workflow](https://github.com/kunitsucom/ddlctl/workflows/go-vuln/badge.svg)](https://github.com/kunitsucom/ddlctl/tree/main)
[![codecov](https://codecov.io/gh/kunitsucom/ddlctl/graph/badge.svg?token=8Jtk2bpTe2)](https://codecov.io/gh/kunitsucom/ddlctl)
[![sourcegraph](https://sourcegraph.com/github.com/kunitsucom/ddlctl/-/badge.svg)](https://sourcegraph.com/github.com/kunitsucom/ddlctl)

## Demo

![ddlctl_demo](https://github.com/kunitsucom/ddlctl/assets/29125616/2ff6bd76-037e-4695-aef1-5ca87a528e07)

## Overview

ddlctl can do the following:  

- Output all RDBMS DDLs
- Generate DDL from tagged Golang source code
- Output differences between the RDBMS and your DDL
- Automated Migration

## TODO

- `generate` subcommand
  - source language
    - [x] Support `go` (beta)
  - dialect
    - [x] Support `mysql` (alpha)
    - [x] Support `postgres` (alpha)
    - [x] Support `cockroachdb` (alpha)
    - [x] Support `spanner` (alpha)
    - [ ] Support `sqlite3`
- `show` subcommand
  - dialect
    - [x] Support `mysql` (beta)
    - [x] Support `postgres` (alpha)
    - [x] Support `cockroachdb` (beta)
    - [x] Support `spanner` (alpha)
    - [ ] Support `sqlite3`
- `diff` subcommand
  - dialect
    - [x] Support `mysql` (alpha)
    - [x] Support `postgres` (alpha)
    - [x] Support `cockroachdb` (alpha)
    - [x] Support `spanner` (alpha)
    - [ ] Support `sqlite3`
- `apply` subcommand
  - dialect
    - [x] Support `mysql` (alpha)
    - [x] Support `postgres` (alpha)
    - [x] Support `cockroachdb` (alpha)
    - [x] Support `spanner` (alpha)
    - [ ] Support `sqlite3`

## Example: `ddlctl generate`

### 1. Prepare your annotated model source code

For example, prepare the following Go code:  

```go
package sample

// User is a user model struct.
//
//pgddl:table public.users
//pgddl:constraint UNIQUE ("username")
//pgddl:index "index_users_username" ON public.users ("username")
type User struct {
    UserID   string `db:"user_id"  pgddl:"TEXT NOT NULL" pk:"true"`
    Username string `db:"username" pgddl:"TEXT NOT NULL"`
    Age      int    `db:"age"      pgddl:"INT  NOT NULL"`
}

// Group is a group model struct.
//
//pgddl:table CREATE TABLE IF NOT EXISTS public.groups
//pgddl:index CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name")
type Group struct {
    GroupID     string `db:"group_id"    pgddl:"TEXT NOT NULL" pk:"true"`
    GroupName   string `db:"group_name"  pgddl:"TEXT NOT NULL"`
    Description string `db:"description" pgddl:"TEXT NOT NULL"`
}
```

Write it to an appropriate file:  

```sh
cat <<"EOF" > /tmp/sample.go
package sample

// User is a user model struct.
//
//pgddl:table public.users
//pgddl:constraint UNIQUE ("username")
//pgddl:index "index_users_username" ON public.users ("username")
type User struct {
    UserID   string `db:"user_id"  pgddl:"TEXT NOT NULL" pk:"true"`
    Username string `db:"username" pgddl:"TEXT NOT NULL"`
    Age      int    `db:"age"      pgddl:"INT  NOT NULL"`
}

// Group is a group model struct.
//
//pgddl:table CREATE TABLE IF NOT EXISTS public.groups
//pgddl:index CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name")
type Group struct {
    GroupID     string `db:"group_id"    pgddl:"TEXT NOT NULL" pk:"true"`
    GroupName   string `db:"group_name"  pgddl:"TEXT NOT NULL"`
    Description string `db:"description" pgddl:"TEXT NOT NULL"`
}
EOF
```

### 2. Generate DDL

Please execute the ddlctl command as follows:  

```console
$ ddlctl generate --dialect postgres --column-tag-go db --ddl-tag-go pgddl --pk-tag-go pk --src /tmp/sample.go --dst /tmp/sample.sql
INFO: 2023/11/16 16:10:39 ddlctl.go:44: source: /tmp/sample.go
INFO: 2023/11/16 16:10:39 ddlctl.go:73: destination: /tmp/sample.sql
```

### 3. Check generated DDL file

Please check the contents of the outputted DDL:  

```sh
cat /tmp/sample.sql
```

content:  

```sql
-- Code generated by ddlctl. DO NOT EDIT.
--

-- source: tmp/sample.go:5
-- User is a user model struct.
--
-- pgddl:table public.users
-- pgddl:constraint UNIQUE ("username")
CREATE TABLE public.users (
    "user_id"  TEXT NOT NULL,
    "username" TEXT NOT NULL,
    "age"      INT  NOT NULL,
    PRIMARY KEY ("user_id"),
    UNIQUE ("username")
);

-- source: tmp/sample.go:7
-- pgddl:index "index_users_username" ON public.users ("username")
CREATE INDEX "index_users_username" ON public.users ("username");

-- source: tmp/sample.go:16
-- Group is a group model struct.
--
-- pgddl:table CREATE TABLE IF NOT EXISTS public.groups
CREATE TABLE IF NOT EXISTS public.groups (
    "group_id"    TEXT NOT NULL,
    "group_name"  TEXT NOT NULL,
    "description" TEXT NOT NULL,
    PRIMARY KEY ("group_id")
);

-- source: tmp/sample.go:17
-- pgddl:index CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name")
CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name");
```

## Example: `ddlctl diff` and `ddlctl apply`

### 1. Prepare your DDL

```sh
cat /tmp/sample.sql
```

content:  

```sql
-- Code generated by ddlctl. DO NOT EDIT.
--

-- source: tmp/sample.go:5
-- User is a user model struct.
--
-- pgddl:table public.users
-- pgddl:constraint UNIQUE ("username")
CREATE TABLE public.users (
    "user_id"  TEXT NOT NULL,
    "username" TEXT NOT NULL,
    "age"      INT  NOT NULL,
    PRIMARY KEY ("user_id"),
    UNIQUE ("username")
);

-- source: tmp/sample.go:7
-- pgddl:index "index_users_username" ON public.users ("username")
CREATE INDEX "index_users_username" ON public.users ("username");

-- source: tmp/sample.go:16
-- Group is a group model struct.
--
-- pgddl:table CREATE TABLE IF NOT EXISTS public.groups
CREATE TABLE IF NOT EXISTS public.groups (
    "group_id"    TEXT NOT NULL,
    "group_name"  TEXT NOT NULL,
    "description" TEXT NOT NULL,
    PRIMARY KEY ("group_id")
);

-- source: tmp/sample.go:17
-- pgddl:index CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name")
CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name");
```

### 2. Check diff from local DDL file to DSN

Please check the differences between the local DDL file and the destination database:

```console
$ ddlctl diff --dialect postgres "postgres://postgres:password@localhost/testdb?sslmode=disable" /tmp/sample.sql
CREATE TABLE public.users (
    "user_id" TEXT NOT NULL,
    "username" TEXT NOT NULL,
    "age" INT NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY ("user_id"),
    CONSTRAINT users_unique_username UNIQUE ("username")
);
CREATE INDEX "index_users_username" ON public.users ("username");
CREATE TABLE IF NOT EXISTS public.groups (
    "group_id" TEXT NOT NULL,
    "group_name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    CONSTRAINT groups_pkey PRIMARY KEY ("group_id")
);
CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name");
```

### 3. Apply DDL

```console
$ ddlctl apply --dialect postgres "postgres://postgres:password@localhost/testdb?sslmode=disable" /tmp/sample.sql --auto-approve

ddlctl will exec the following DDL queries:

-- 8< --

CREATE TABLE public.users (
    "user_id" TEXT NOT NULL,
    "username" TEXT NOT NULL,
    "age" INT NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY ("user_id"),
    CONSTRAINT users_unique_username UNIQUE ("username")
);
CREATE INDEX "index_users_username" ON public.users ("username");
CREATE TABLE IF NOT EXISTS public.groups (
    "group_id" TEXT NOT NULL,
    "group_name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    CONSTRAINT groups_pkey PRIMARY KEY ("group_id")
);
CREATE UNIQUE INDEX "index_groups_group_name" ON public.groups ("group_name");


-- >8 --

Do you want to apply these DDL queries?
  ddlctl will exec the DDL queries described above.
  Only 'yes' will be accepted to approve.

Enter a value: yes (via --auto-approve option)

executing...
done
```

### 4. (Optional) Edit DDL and apply

```diff
 -- pgddl:table public.users
 -- pgddl:constraint UNIQUE ("username")
 CREATE TABLE public.users (
     "user_id"     TEXT NOT NULL,
     "username"    TEXT NOT NULL,
     "age"         INT  NOT NULL,
+    "description" TEXT NOT NULL,
     PRIMARY KEY ("user_id"),
     UNIQUE ("username")
 );
```

apply:

```console
$ ddlctl apply --dialect postgres "postgres://postgres:password@localhost/testdb?sslmode=disable" /tmp/sample.sql --auto-approve

ddlctl will exec the following DDL queries:

-- 8< --

-- -
-- +"description" TEXT NOT NULL
ALTER TABLE public.users ADD COLUMN "description" TEXT NOT NULL;


-- >8 --

Do you want to apply these DDL queries?
  ddlctl will exec the DDL queries described above.
  Only 'yes' will be accepted to approve.

Enter a value: yes (via --auto-approve option)

executing...
done
```

## Installation

### pre-built binary

```bash
VERSION=v0.0.8

# download
curl -fLROSs https://github.com/kunitsucom/ddlctl/releases/download/${VERSION}/ddlctl_${VERSION}_darwin_arm64.zip

# unzip
unzip -j ddlctl_${VERSION}_darwin_arm64.zip '*/ddlctl'
```

### go install

```bash
go install github.com/kunitsucom/ddlctl/cmd/ddlctl@v0.0.8
```

## Usage

### `ddlctl`

```console
$ ddlctl --help
Usage:
    ddlctl [options]

Description:
    ddlctl is a tool for control RDBMS DDL.

sub commands:
    version: show version
    generate: generate DDL from source (file or directory) to destination (file or directory).
    show: show DDL from DSN like `SHOW CREATE TABLE`.
    diff: diff DDL from <before DDL source> to <after DDL source>.
    apply: apply DDL from <DDL source> to <DSN to apply>.

options:
    --trace (env: DDLCTL_TRACE, default: false)
        trace mode enabled
    --debug (env: DDLCTL_DEBUG, default: false)
        debug mode
    --help (default: false)
        show usage
```

### `ddlctl generate`

```console
$ ddlctl generate --help
Usage:
    ddlctl generate [options] --dialect <DDL dialect> --src <source> --dst <destination>

Description:
    generate DDL from source (file or directory) to destination (file or directory).

options:
    --lang (env: DDLCTL_LANGUAGE, default: go)
        programming language to generate DDL
    --dialect (env: DDLCTL_DIALECT, default: )
        SQL dialect to generate DDL
    --column-tag-go (env: DDLCTL_COLUMN_TAG_GO, default: db)
        column annotation key for Go struct tag
    --ddl-tag-go (env: DDLCTL_DDL_TAG_GO, default: ddlctl)
        DDL annotation key for Go struct tag
    --pk-tag-go (env: DDLCTL_PK_TAG_GO, default: pk)
        primary key annotation key for Go struct tag
    --src (env: DDLCTL_SOURCE, default: /dev/stdin)
        source file or directory
    --dst (env: DDLCTL_DESTINATION, default: /dev/stdout)
        destination file or directory
    --help (default: false)
        show usage
```

### `ddlctl show`

```console
$ ddlctl show --help
Usage:
    ddlctl show --dialect <DDL dialect> <DSN>

Description:
    show DDL from DSN like `SHOW CREATE TABLE`.

options:
    --dialect (env: DDLCTL_DIALECT, default: )
        SQL dialect to generate DDL
    --help (default: false)
        show usage
```

### `ddlctl diff`

```console
$ ddlctl diff --help
Usage:
    ddlctl diff [options] --dialect <DDL dialect> <before DDL source> <after DDL source>

Description:
    diff DDL from <before DDL source> to <after DDL source>.

options:
    --lang (env: DDLCTL_LANGUAGE, default: go)
        programming language to generate DDL
    --dialect (env: DDLCTL_DIALECT, default: )
        SQL dialect to generate DDL
    --column-tag-go (env: DDLCTL_COLUMN_TAG_GO, default: db)
        column annotation key for Go struct tag
    --ddl-tag-go (env: DDLCTL_DDL_TAG_GO, default: ddlctl)
        DDL annotation key for Go struct tag
    --pk-tag-go (env: DDLCTL_PK_TAG_GO, default: pk)
        primary key annotation key for Go struct tag
    --help (default: false)
        show usage
```

### `ddlctl apply`

```console
$ ddlctl apply --help
Usage:
    ddlctl apply [options] --dialect <DDL dialect> <DSN to apply> <DDL source>

Description:
    apply DDL from <DDL source> to <DSN to apply>.

options:
    --lang (env: DDLCTL_LANGUAGE, default: go)
        programming language to generate DDL
    --dialect (env: DDLCTL_DIALECT, default: )
        SQL dialect to generate DDL
    --column-tag-go (env: DDLCTL_COLUMN_TAG_GO, default: db)
        column annotation key for Go struct tag
    --ddl-tag-go (env: DDLCTL_DDL_TAG_GO, default: ddlctl)
        DDL annotation key for Go struct tag
    --pk-tag-go (env: DDLCTL_PK_TAG_GO, default: pk)
        primary key annotation key for Go struct tag
    --auto-approve (env: DDLCTL_AUTO_APPROVE, default: false)
        auto approve
    --help (default: false)
        show usage
```
