#!/usr/bin/env gisp

(begin
  (display "Demonstrating call/cc\n")
  (define saved #f)
  (define result
    (call/cc (lambda (k)
               (set! saved k)
               "initial return")))
  (display "First result: ")
  (display result)
  (newline)
  (if (stringp result)
      (begin
        (display "Invoking continuation with 42\n")
        (saved 42))
      (begin
        (display "Continuation produced: ")
        (display result)
        (newline)
        result)))
