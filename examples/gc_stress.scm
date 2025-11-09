#!/usr/bin/env guile
!#
;; GC stress benchmark rewritten in Scheme for Guile.
;; Mirrors the allocation patterns from the Gisp Go-like script.

(define short-lived-iterations 20000)
(define short-lived-size 128)

(define mixed-iterations 6000)
(define mixed-size 256)
(define mixed-survivors 512)
(define mixed-refresh-interval 10)

(define closure-iterations 18000)
(define closure-env-size 32)
(define closure-keep-interval 25)

(define symbol-burst-count 15000)
(define symbol-prefix "gc-stress-")

(define (make-list size)
  (let loop ((i 0) (head '()))
    (if (= i size)
        head
        (loop (+ i 1) (cons i head)))))

(define (trim-list lst limit)
  (let loop ((cursor lst) (count 0) (acc '()))
    (if (or (null? cursor) (>= count limit))
        (reverse acc)
        (loop (cdr cursor) (+ count 1) (cons (car cursor) acc)))))

(define (churn-short-lived iterations size)
  (let loop ((i 0) (checksum 0))
    (if (= i iterations)
        checksum
        (let ((temp (make-list size)))
          (loop (+ i 1) (+ checksum (length temp)))))))

(define (preload-survivors target size)
  (let loop ((count 0) (acc '()))
    (if (= count target)
        acc
        (loop (+ count 1) (cons (make-list size) acc)))))

(define (churn-mixed-lifetimes iterations size survivor-limit refresh-interval)
  (let loop ((step 0)
             (checksum 0)
             (survivors (preload-survivors survivor-limit size))
             (next-refresh 0))
    (if (= step iterations)
        (+ checksum (length survivors))
        (let* ((ephemeral (make-list size))
               (checksum* (+ checksum (length ephemeral)))
               (attach-now? (and (> refresh-interval 0) (= step next-refresh))))
          (if attach-now?
              (let ((new-survivors (trim-list (cons ephemeral survivors) survivor-limit)))
                (loop (+ step 1) checksum* new-survivors (+ next-refresh refresh-interval)))
              (loop (+ step 1) checksum* survivors next-refresh))))))

(define (sum-list lst)
  (let loop ((cursor lst) (total 0))
    (if (null? cursor)
        total
        (loop (cdr cursor) (+ total (car cursor))))))

(define (churn-closures iterations env-size keep-interval)
  (let loop ((i 0)
             (checksum 0)
             (keepers '())
             (next-keep 0))
    (if (= i iterations)
        (+ checksum (length keepers))
        (let* ((payload (make-list env-size))
               (closure (lambda (x)
                          (+ (sum-list payload) x))))
          (if (and (> keep-interval 0) (= i next-keep))
              (loop (+ i 1) checksum (cons closure keepers) (+ next-keep keep-interval))
              (loop (+ i 1) (+ checksum (closure i)) keepers next-keep))))))

(define (intern-symbol-burst count prefix)
  (let loop ((i 0))
    (if (= i count)
        count
        (begin
          (string->symbol (string-append prefix (number->string i)))
          (loop (+ i 1))))))

(define (run-phase name thunk)
  (display "== ")
  (display name)
  (newline)
  (let ((result (thunk)))
    (display "Result checksum: ")
    (display result)
    (newline)
    (newline)))

(define (main)
  (display "Guile GC stress benchmark starting.")
  (newline)
  (newline)

  (run-phase "Short-lived list churn"
             (lambda ()
               (churn-short-lived short-lived-iterations short-lived-size)))

  (run-phase "Mixed lifetimes with survivor pool"
             (lambda ()
               (churn-mixed-lifetimes
                mixed-iterations
                mixed-size
                mixed-survivors
                mixed-refresh-interval)))

  (run-phase "Closure allocation and retention"
             (lambda ()
               (churn-closures
                closure-iterations
                closure-env-size
                closure-keep-interval)))

  (run-phase "Symbol interning burst"
             (lambda ()
               (intern-symbol-burst symbol-burst-count symbol-prefix)))

  (display "Benchmark run complete. Use external tooling for timing/GC stats.")
  (newline))

(main)

