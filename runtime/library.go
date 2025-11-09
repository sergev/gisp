package runtime

var preludeForms = []string{
	`
(define-macro (and . args)
  (if (nullp args)
      #t
      (if (nullp (cdr args))
          (car args)
          (list 'if (car args)
                (cons 'and (cdr args))
                '#f))))
`,
	`
(define-macro (or . args)
  (if (nullp args)
      #f
      (let ((rest (cdr args)))
        (if (nullp rest)
            (car args)
            (let ((sym (gensym)))
              (list 'let (list (list sym (car args)))
                    (list 'if sym sym (cons 'or rest))))))))
`,
}
