import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"

export default function StudentIndex() {
  return (
    <div className="p-8">
      <Card>
        <CardHeader>
          <CardTitle>Student area</CardTitle>
        </CardHeader>
        <CardContent>
          <p>Welcome to the student dashboard.</p>
        </CardContent>
      </Card>
    </div>
  )
}
