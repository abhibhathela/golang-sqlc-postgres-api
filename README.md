# this me first time writing code in go lang \*_pardon any mistakes_

to run the appliction

1. create env file

```
cp .env.sample .env
```

2. add relevant env variables

3. create db `rewards`
4. run migration

```
make migrate-up
```

5. run application

```
make run-live
```

6. to test application

```
make run-test
```
