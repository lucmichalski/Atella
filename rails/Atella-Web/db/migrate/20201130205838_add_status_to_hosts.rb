class AddStatusToHosts < ActiveRecord::Migration[6.0]
  def change
    add_column :hosts, :status, :boolean
  end
end
