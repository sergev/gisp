;; sierpinski.scm
(define (make-sierpinski n)
  (if (= n 0)
      '("*")
      (let* ((prev (make-sierpinski (- n 1)))
             (space (make-string (string-length (first prev)) #\space)))
        (append
         (map (lambda (line) (string-append space line space)) prev)
         (map (lambda (line) (string-append line " " line)) prev)))))

(define (print-sierpinski n)
  (for-each display-line (make-sierpinski n)))

(define (display-line line)
  (display line)
  (newline))

;; Example:
(print-sierpinski 4)
