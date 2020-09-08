class ApplicationController < ActionController::Base
  include ApplicationHelper
  include AtellaMainHelper
  def initialize
    super
    @settings = Rails.application.config.atella
    @atella = TOML.load_file(@settings["atella"]["atellaConfig"])
  end
end
