let () =
  Test_parser.run ();
  Test_utils.run ();
  Test_theme.run ();
  Test_directives.run ();
  Test_candidate.run ();
  Harness.finish ()
