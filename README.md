# Pete
a cli tool for persist query option generation/matainance

## persist.ql query helper

protoc-gen-persist is a protoc plugin that generates a persistence layer from proto => sql/spanner database

Given a filled out persist.ql option, protoc-gen-persist will generate a library of functions that can communicate with a backend given only a protobuf input.



This while powerful, is hard to explain in a proto file.  Usually a service ends up with a ql option looking like this
```js
option (persist.ql) = {
        queries: [{
            name: "create_users_table",
            query: ["CREATE TABLE users(id integer PRIMARY KEY, name VARCHAR(50), friends BYTEA,",
                    "created_on VARCHAR(50), id2 SMALLINT)"],
            pm_strategy: "$",
            in: ".pb.Empty",
            out: ".pb.Empty",
        }, {
            name: "insert_users",
            query: ["INSERT INTO users (id, name, friends, created_on, id2) VALUES (@id, @name, @friends, @created_on, @id2)"],
            pm_strategy: "$",
            in: ".pb.User",
            out: ".pb.Empty",
        }, {
            name: "get_all_users",
            query: ["SELECT id, name, friends, created_on, id2 FROM users"],
            in: ".pb.Empty",
            out: ".pb.User",
        }, {
            name: "select_user_by_id",
            query: ["SELECT id, name, friends, created_on, id2 FROM users WHERE id = @id"],
            pm_strategy: "$",
            in: ".pb.User",
            out: ".pb.User",
        }, {
            name: "update_user_name",
            query: ["Update users set name = @name WHERE id = @id ",
                    "RETURNING id, name, friends, created_on"],
            pm_strategy: "$",
            in: ".pb.User",
            out: ".pb.User",
        }, {
            name: "update_name_to_foo",
            query: ["Update users set name = 'foo' WHERE id = @id"],
            pm_strategy: "$",
            in: ".pb.User",
            out: ".pb.Empty",
        }, {
            query: ["SELECT id, name, friends, created_on, id2 FROM users WHERE name = ANY(@names)"],
            pm_strategy: "$",
            name: "get_friends",
            in: ".pb.FriendsReq",
            out: ".pb.User",
        },{
            query: ["drop table users"],
            name: "drop",
            in: ".pb.Empty",
            out: ".pb.Empty",
        }]
    };
 ```

**Writing this by hand is tedious.**

-  Missing commas, or other syntax errors do not report the line number, so the bigger the service, the more
 digging has to be done.
- Wrapping every line of the query with `padding + "SELECT...",` is really annoying
- Inside an array must follow js rules on comma placement, but outside of arrays it does not matter.
    This, combined with the fact that the best place to split a line is often after a comma, confuses
    new programmers, because you often see something like this 
```
{
    query: [
        "SELECT",
            "id,", 
            "name,",
            "friends,",
            "created_on,",
            "id2", // <<-- pattern was ,",  now is ",
        "FROM users",
        "WHERE name = ANY(@names)" // <-- no comma allowed
    ],
    pm_strategy: "$",
    name: "get_friends",
    in: ".pb.FriendsReq",
    out: ".pb.User",   // <<-- but this one is ok
}
```
- The extra characters and extra comma often trick your eye into writing a bad query. Bad queries
    can only be caught by running the code to test it. Time is wasted doing it wrong.
- Refactoring a query becomes a chore.  You can't press enter on a line without having to add ", to the last line and "
    at the begining of the next.  
- You have to repeat yourself often. `in` and `out` options require full qualified paths, even if the message is local.
    `pm_strategy:` almost never will change on a query by query basis for most services, yet you must write it for every query



Pete is a cli tool for snipping out a (persist.ql) option in a `.proto` file, and replacing it's with uglified code, with code  from a much prettier input file.


## testfile:
```
insert_user
INSERT INTO users (id, name, friends, created_on, id2) VALUES (@id, @name, @friends, @created_on, @id2)
in: User
out: Empty

get_all_users
SELECT
    id,
    name,
    friends,
    created_on,
    id2
FROM users
in: Empty
out: User
```

```sh
$ pete -h
Usage:
  pete [flags]

Flags:
      --config string    config file (default is $HOME/.pete.yaml)
  -d, --deli string      the delimiter to use (default "\n\n")
  -h, --help             help for pete
  -i, --input string     file to parse
  -l, --linepad string   the padding string for each line defaults to 4 spaces (default "    ")
  -o, --output string    file to write to
  -p, --prefix string    the package prefix for your in and out types
  ```
`$ pete -i testfile -o user.proto -p .pb`

will replace the current persist.ql queries with the following:
```
{
    name: "insert_users",
    query: [
        "INSERT INTO users (id, name, friends, created_on, id2) VALUES (@id, @name, @friends, @created_on, @id2)"
    ],
    pm_strategy: "$",
    in: ".pb.User",
    out: ".pb.Empty",
},
{
    name: "get_all_users",
    query: [
        "SELECT",
            "id,",
            "name,",
            "friends,",
            "created_on,",
            "id2",
        "FROM users"
    ],
    pm_strategy: "$",
    in: ".pb.Empty",
    out: ".pb.User",
}
```

no more comma hunting, no more syntax errors, no more having to write more than is needed.

**Rules**
- query name must always be on the first line after the delimiter
- in, and out must be prefixed with `in: TYPE_NAME` and `out: TYPE_NAME`
- pete refuses to override a `in:` or `out:` option if it has `.`'s in the string. It assumes it must be
    fully qualified


## Roadmap
- read from a proto file into a .pete file so I can change all the other services to this format not by hand.
- be able to snip in/out single queries by name.