class AddIsMasterToHosts < ActiveRecord::Migration[6.0]
  def change
    add_column :hosts, :is_master, :boolean, null: false, default: false
  end
end
