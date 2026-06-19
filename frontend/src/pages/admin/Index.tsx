import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"

export default function AdminIndex() {
  return (
    <div className="p-8">
      <Card>
        <CardHeader>
          <CardTitle>Admin area</CardTitle>
        </CardHeader>
        <CardContent>
          <p>Welcome to the admin dashboard.</p>
        </CardContent>
      </Card>
    </div>
  )
}
