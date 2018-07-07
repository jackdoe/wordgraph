# wordgraph

![screencast](https://raw.githubusercontent.com/jackdoe/wordgraph/master/tty.gif)

Build a graph of flow of words like "a -> b -> c -> d"

I trained it on 15000 books from project guttenberg, and cutoff the
edges that are below 75th percentile. (you can find this index in computed_index/)


## /flow

for example if you look at 'this is an apple' vs 'this is a apple':

```
$ curl -s -d '{"text":"this is a apple"}' http://localhost:8080/flow | json_pp
{
   "Items" : [
      {
         "Word" : "this",
         "Score" : 2
      },
      {
         "Word" : "is",
         "Score" : 3
      },
      {
         "Score" : 3,
         "Word" : "a"
      },
      {
         "Score" : 0,
         "Word" : "apple"
      }
   ]
}

$ curl -s -d '{"text":"this is an apple"}' http://localhost:8080/flow | json_pp
{
   "Items" : [
      {
         "Word" : "this",
         "Score" : 2
      },
      {
         "Score" : 4,
         "Word" : "is"
      },
      {
         "Word" : "an",
         "Score" : 4
      },
      {
         "Score" : 3,
         "Word" : "apple"
      }
   ]
}

```

## /query

```

curl -d '{"text":"this is an apple","percentile":99.5}' http://localhost:8080/query | json_pp

{
   "Processed" : {
      "an" : {
         "Flows" : [
            {
               "Words" : [
                  "if",
                  "an",
                  "individual"
               ],
               "Count" : 33314
            },
            {
               "Words" : [
                  "half",
                  "an",
                  "hour"
               ],
               "Count" : 19481
            },
            {
               "Count" : 11087,
               "Words" : [
                  "with",
                  "an",
                  "electronic"
               ]
            },
            {
               "Words" : [
                  "for",
                  "an",
                  "instant"
               ],
               "Count" : 9665
            },
            {
               "Count" : 8714,
               "Words" : [
                  "of",
                  "an",
                  "hour"
               ]
            },
            {
               "Words" : [
                  "of",
                  "an",
                  ...
```

## FIXME

* the tokenizer is not good
* add part-of-speech tagging and use that as alternative graph

## LICENSE

MIT
