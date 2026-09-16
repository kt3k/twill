let () =
  Test_parser.run ();
  Test_utils.run ();
  Test_theme.run ();
  Test_directives.run ();
  Test_candidate.run ();
  Test_builtin_variants.run ();
  Test_builtin_utilities.run ();
  Test_compile.run ();
  Test_extract.run ();
  Test_scanner.run ();
  Test_cli.run ();
  Harness.finish ()
