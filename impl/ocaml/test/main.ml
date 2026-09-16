let () =
  Test_parser.run ();
  Test_utils.run ();
  Test_theme.run ();
  Test_directives.run ();
  Test_candidate.run ();
  Test_builtin_variants.run ();
  Test_builtin_utilities.run ();
  Test_compile.run ();
  Harness.finish ()
