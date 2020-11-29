module ApplicationHelper
  def wrap_master_host(addr, name, sec)
    status = Hash.new
    status["status"] = "Failed"
    status["version"] = "unknown"
    if sec["security"].nil?
      return status.to_json
    end
    if sec["security"]["code"].nil?
      return status.to_json
    end
    begin
      s = TCPSocket.open(addr, 5223)
      s.puts("auth #{sec["security"]["code"]}")
      loop do
        line = s.gets
        break if line.nil?
        l = line.rstrip.split
        case (l[0])
          when "+OK"
            break if l.length < 2
            case (l[1]) 
              when "bye!"
                break
              when "ack"
                break if l.length < 3
                case (l[2])
                  when "auth"
                    s.puts("export master")
                  when "master"
                    break if l.length < 4
                    status["status"] = JSON.parse(l[3])
                    s.puts("get version")
                  when "version"
                    break if l.length < 4
                    status["version"] = l[3]
                    unless status["version"].nil?
                      status["version"].rstrip!
                    end
                    s.puts("exit")
                  else
                    break
                end
              else
                break
            end
          else
            break
        end
      end
      s.close
    rescue => v
      status["status"] = v
    end
    return status.to_json
  end

  def processVector(v)
    _s = JSON.parse(v)
    s = _s["status"]
    # s.each do |i|
    #   print "#{i}\n"
    # end
    return v
  end
end