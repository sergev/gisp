package runtime

var preludeForms = []string{
	`
(define-macro (and . args)
  (if (nullp args)
      #t
      (if (nullp (rest args))
          (first args)
          (list 'if (first args)
                (cons 'and (rest args))
                '#f))))
`,
	`
(define-macro (or . args)
  (if (nullp args)
      #f
      (let ((rst (rest args)))
        (if (nullp rst)
            (first args)
            (let ((sym (gensym)))
              (list 'let (list (list sym (first args)))
                    (list 'if sym sym (cons 'or rst))))))))
`,
}
