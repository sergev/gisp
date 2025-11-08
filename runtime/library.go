package runtime

var preludeForms = []string{
	`
(define-macro (and . args)
  (if (null? args)
      #t
      (if (null? (cdr args))
          (car args)
          (list 'if (car args)
                (cons 'and (cdr args))
                '#f))))
`,
	`
(define-macro (or . args)
  (if (null? args)
      #f
      (let ((rest (cdr args)))
        (if (null? rest)
            (car args)
            (let ((sym (gensym)))
              (list 'let (list (list sym (car args)))
                    (list 'if sym sym (cons 'or rest))))))))
`,
}
