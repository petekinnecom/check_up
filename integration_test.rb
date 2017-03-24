require "minitest/autorun"
require "open3"
require "securerandom"

class IntegrationTest < Minitest::Test

  def setup
    raise "Compile the binary first" unless File.exist?("./_build/check_up_test")
  end

  def test_integration__success
    Tempfile.open do |file|
      file.puts <<~YAML
        services:
          - name: service_1
            command: exit 0
          - name: service_2
            command: exit 0
      YAML
      file.flush

      out, status = Open3.capture2e("./_build/check_up_test --file #{file.path}")
      assert status == 0
      assert out.empty?, "no news is good news"
    end
  end

  def test_integration__fail
    Tempfile.open do |file|
      file.puts <<~YAML
        services:
          - name: service_1
            command: exit 0
          - name: service_2
            command: exit 1
      YAML
      file.flush

      out, status = Open3.capture2e("./_build/check_up_test --file #{file.path}")
      assert status != 0
      assert out.match(/service_2 | down/)
    end
  end

  def test_integration__wait
    random_file = File.join("/tmp", SecureRandom.uuid)

    Tempfile.open do |file|
      file.puts <<~YAML
        services:
          - name: service_1
            command: test -f #{random_file}
            interval: 1
      YAML
      file.flush

      # ensure our preconditions
      out, status = Open3.capture2e("./_build/check_up_test --file #{file.path}")
      refute status == 0
      assert out.match(/service_1 | down/)

      thread = Thread.new {
        out, status = Open3.capture2e("./_build/check_up_test --file #{file.path} --wait --verbose")
      }

      sleep 1 # Magic number to let the executable spin up

      File.open(random_file, "w") {}
      thread.join
      assert status == 0
      expected_lines = [
        "service_1 | trying",
        "service_1 | test -f #{random_file}",
        "service_1 | exit status 1",
        "service_1 | down",
        "retrying check up",
        "service_1 | trying",
        "service_1 | test -f #{random_file}",
        "service_1 | up",
      ].each do |line|
        assert out.match(line), "expected to find #{line} in:\n#{out}"
      end
    end
  end
end
