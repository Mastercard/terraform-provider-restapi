# A fake API object server you can start from the command line

Fakeserver is used by the testing suite and is wrapped by a simple CLI tool that allows you to start it outside of the test suite.

There are only a few options for `fakeservercli`:
`-port` (int) - the port on 127.0.0.1 the fakeserver will bind to. Defaults to 8080
`-debug` - Will produce verbose information to STDOUT on requests and responses
`-static_dir` - When set, will serve files in this directory under the path /static/[name_of_file]

Once running, fakeserver is expecting you to populate it with data that means whatever you like it to mean.

There are a few things to know:
 - All objects are at `/api/objects/{id}`
 - `id` is a required field. It is how `fakeserver` finds the objects
 - All objects are internally represented and returned as strings for both keys and values
 - A GET to an ID will print the JSON representation of the object
 - A POST to `/api/objects` will save the object in memory and return the JSON representation of the object
 - A PUT to `/api/objects/{id}` will update the object at that location with the data sent (fields removed are not preserved)
 - A DELETE to `/api/objects/{id}` will remove the object at that ID from memory

### Populate the fakeserver
```
curl 127.0.0.1:8080/api/objects -X POST -d '{ "id": "1", "name": "Foo"}'
curl 127.0.0.1:8080/api/objects -X POST -d '{ "id": "2", "name": "Bar"}'
curl 127.0.0.1:8080/api/objects -X POST -d '{ "id": "3", "name": "Baz"}'
```

### Rename an object
This example changes the name of 'Baz' to 'Biz'
```
curl 127.0.0.1:8080/api/objects/3 -X PUT -d '{ "id": "3", "name": "Biz"}'
```

### Delete an object
This example just deletes the object with id=3
```
curl 127.0.0.1:8080/api/objects/3 -X DELETE
```
